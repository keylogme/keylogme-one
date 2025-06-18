ifneq (,$(wildcard ../.env))
    include ../.env
    export
endif

dev:
	@go build -o ../bin/keylogme-one-dev ./main.go
	sudo ../bin/keylogme-one-dev ${API_KEY_DEV} ${ENDPOINT_DEV}
run:
	# @go build -o ../bin/keylogme-one ./main.go
	sudo ../bin/keylogme-one ${API_KEY_PROD} ${ENDPOINT_PROD}

test:
	@go test ../... -timeout=30s 
