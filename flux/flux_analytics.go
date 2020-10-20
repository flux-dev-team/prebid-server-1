package flux

import (
	"fmt"
	"github.com/prebid/prebid-server/analytics"
	"go.uber.org/zap"
)

type FluxAnalytics struct {
	logging *zap.Logger
}

func NewFluxAnalytics() *FluxAnalytics {
	fmt.Println("init NewFluxAnalytics()")
	logger, _ := zap.NewProduction()
	return &FluxAnalytics{
		logging: logger,
	}
}

func (f *FluxAnalytics) LogAuctionObject(vo *analytics.AuctionObject) {
	f.logging.Info("fluxLog", zap.Any("LogAuctionObject", vo))
	return
}

func (f *FluxAnalytics) LogVideoObject(vo *analytics.VideoObject) {
	f.logging.Info("fluxLog", zap.Any("LogVideoObject", vo))
	return
}

func (f *FluxAnalytics) LogCookieSyncObject(vo *analytics.CookieSyncObject) {
	f.logging.Info("fluxLog", zap.Any("LogCookieSyncObject", vo))
	return
}

func (f *FluxAnalytics) LogSetUIDObject(vo *analytics.SetUIDObject) {
	f.logging.Info("fluxLog", zap.Any("LogSetUIDObject", vo))
	return
}

func (f *FluxAnalytics) LogAmpObject(vo *analytics.AmpObject) {
	f.logging.Info("fluxLog", zap.Any("LogAmpObject", vo))
	return
}

func (f *FluxAnalytics) LogNotificationEventObject(vo *analytics.NotificationEvent) {
	f.logging.Info("fluxLog", zap.Any("NotificationEvent", vo))
	return
}
