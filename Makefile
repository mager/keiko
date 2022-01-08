
dev:
	go mod tidy && go run main.go

test:
	go test ./...

build:
	gcloud builds submit --tag gcr.io/floor-report-327113/keiko

deploy:
	gcloud run deploy keiko \
		--image gcr.io/floor-report-327113/keiko \
		--platform managed

ship:
	make test && make build && make deploy
