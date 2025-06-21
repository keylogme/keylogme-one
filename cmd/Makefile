ifneq (,$(wildcard ../.env))
    include ../.env
    export
endif

dev:
	@go build -o ../bin/keylogme-one-dev ./main.go
	KEYLOGME_ONE_API_KEY=${API_KEY_DEV} sudo -E ../bin/keylogme-one-dev 
run:
	@go build -o ../bin/keylogme-one ./main.go
	KEYLOGME_ONE_API_KEY=${API_KEY_PROD} sudo -E ../bin/keylogme-one-dev 

test:
	@go test ../... -timeout=30s 
