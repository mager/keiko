# keiko

## Google Cloud Setup

- `gcloud projects create floor-report` - to create a new project
- `gcloud builds submit --tag gcr.io/floor-report-327113/keiko` to build and submit to Google Container Registry
- `gcloud run deploy keiko --image gcr.io/floor-report-327113/keiko --platform managed` to deploy to Cloud Run
