# keiko

Backend for https://floor.report.

## Google Cloud Setup

- `gcloud projects create floor-report` - to create a new project
- `gcloud builds submit --tag gcr.io/floorreport/keiko` to build and submit to Google Container Registry
- `gcloud run deploy keiko --image gcr.io/floorreport/keiko --platform managed` to deploy to Cloud Run
