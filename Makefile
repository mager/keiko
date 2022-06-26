
dev:
	go mod tidy && go run main.go

test:
	go test ./...

build:
	gcloud builds submit --tag gcr.io/floorreport/keiko

deploy:
	gcloud run deploy keiko \
		--image gcr.io/floorreport/keiko \
		--platform managed

ship:
	make test && make build && make deploy
