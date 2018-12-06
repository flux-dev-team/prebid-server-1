package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"
)

func TestCookieSyncNoCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsCookie(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "pubmatic"]}`, nil, false, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRPreventsBidders(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic", "lifestreet"],"gdpr_consent":"BOONs2HOONs2HABABBENAGgAAAAPrABACGA"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetSyncer(&config.Configuration{ExternalURL: "someurl.com"}),
	})
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"lifestreet"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRIgnoredIfZero(t *testing.T) {
	rr := doPost(`{"gdpr":0,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "pubmatic"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestGDPRConsentRequired(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
	assert.Equal(t, rr.Header().Get("Content-Type"), "text/plain; charset=utf-8")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "gdpr_consent is required if gdpr=1\n", rr.Body.String())
}

func TestCookieSyncHasCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, map[string]string{
		"adnxs":           "1234",
		"audienceNetwork": "2345",
	}, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "ok", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that an empty bidders array returns no syncs
func TestCookieSyncEmptyBidders(t *testing.T) {
	rr := doPost(`{"bidders": []}`, nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

// Make sure that all syncs are returned if "bidders" isn't a key
func TestCookieSyncNoBidders(t *testing.T) {
	rr := doPost("{}", nil, true, syncersForTest())
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork", "lifestreet", "pubmatic"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func TestCookieSyncNoCookiesBrokenGDPR(t *testing.T) {
	rr := doConfigurablePost(`{"bidders":["appnexus", "audienceNetwork", "random"],"gdpr_consent":"GLKHGKGKKGK"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{}, config.GDPR{UsersyncIfAmbiguous: true})
	assert.Equal(t, rr.Header().Get("Content-Type"), "application/json; charset=utf-8")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.ElementsMatch(t, []string{"appnexus", "audienceNetwork"}, parseSyncs(t, rr.Body.Bytes()))
	assert.Equal(t, "no_cookie", parseStatus(t, rr.Body.Bytes()))
}

func doPost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer) *httptest.ResponseRecorder {
	return doConfigurablePost(body, existingSyncs, gdprHostConsent, gdprBidders, config.GDPR{})
}

func doConfigurablePost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer, cfgGDPR config.GDPR) *httptest.ResponseRecorder {
	endpoint := testableEndpoint(mockPermissions(gdprHostConsent, gdprBidders), cfgGDPR)
	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)
	req, _ := http.NewRequest("POST", "/cookie_sync", strings.NewReader(body))
	if len(existingSyncs) > 0 {
		pcs := usersync.NewPBSCookie()
		for bidder, uid := range existingSyncs {
			pcs.TrySync(bidder, uid)
		}
		req.AddCookie(pcs.ToHTTPCookie(90 * 24 * time.Hour))
	}

	rr := httptest.NewRecorder()
	endpoint(rr, req, nil)
	return rr
}

func testableEndpoint(perms gdpr.Permissions, cfgGDPR config.GDPR) httprouter.Handle {
	return NewCookieSyncEndpoint(syncersForTest(), &config.Configuration{GDPR: cfgGDPR}, perms, &metricsConf.DummyMetricsEngine{}, analyticsConf.NewPBSAnalytics(&config.Analytics{}))
}

func syncersForTest() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus: appnexus.NewAppnexusSyncer(&config.Configuration{ExternalURL: "someurl.com"}),
		openrtb_ext.BidderFacebook: audienceNetwork.NewFacebookSyncer(&config.Configuration{Adapters: map[string]config.Adapter{
			strings.ToLower(string(openrtb_ext.BidderFacebook)): {
				UserSyncURL: "https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID",
			},
		}}),
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetSyncer(&config.Configuration{ExternalURL: "anotherurl.com"}),
		openrtb_ext.BidderPubmatic:   pubmatic.NewPubmaticSyncer(&config.Configuration{ExternalURL: "thaturl.com"}),
	}
}

func parseStatus(t *testing.T, responseBody []byte) string {
	t.Helper()
	val, err := jsonparser.GetString(responseBody, "status")
	if err != nil {
		t.Fatalf("response.status was not a string. Error was %v", err)
	}
	return val
}

func parseSyncs(t *testing.T, response []byte) []string {
	t.Helper()
	var syncs []string
	jsonparser.ArrayEach(response, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType != jsonparser.Object {
			t.Errorf("response.bidder_status contained unexpected element of type %v.", dataType)
		}
		if val, err := jsonparser.GetString(value, "bidder"); err != nil {
			t.Errorf("response.bidder_status[?].bidder was not a string. Value was %s", string(value))
		} else {
			syncs = append(syncs, val)
		}
	}, "bidder_status")
	return syncs
}

func mockPermissions(allowHost bool, allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer) gdpr.Permissions {
	return &gdprPerms{
		allowHost:      allowHost,
		allowedBidders: allowedBidders,
	}
}

type gdprPerms struct {
	allowHost      bool
	allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer
}

func (g *gdprPerms) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return g.allowHost, nil
}

func (g *gdprPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	_, ok := g.allowedBidders[bidder]
	return ok, nil
}

func (g *gdprPerms) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return true, nil
}
