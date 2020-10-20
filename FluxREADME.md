# README
Please implement the original functionality in ./flux

Dose not git commit secret file, config file and so on...
but original pbs.yaml has flux private repo. ask flux developer


# Health Check
http://localhost:8000/status
```
curl -i http://localhost:8000/status

HTTP/1.1 200 OK
Cache-Control: no-cache, no-store, must-revalidate
Expires: 0
Pragma: no-cache
Vary: Origin
Date: Wed, 21 Oct 2020 14:57:31 GMT
Content-Length: 25
Content-Type: text/plain; charset=utf-8

Hello World From Flux inc% 
```


# SET UP

## local
cp ../pbs.yaml ./


## prod
cp ../pbs.yaml ./
export EVN=prod
