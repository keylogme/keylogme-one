ifneq (,$(wildcard ./.env))
    include .env
    export
endif

dev:
	@go build -o ./gokeny_dev ./main.go
	sudo ./gokeny_dev ${API_KEY_DEV} ${ENDPOINT_DEV}
run:
	@go build -o ./gokeny_prod ./main.go
	sudo ./gokeny_prod ${API_KEY_PROD} ${ENDPOINT_PROD}

