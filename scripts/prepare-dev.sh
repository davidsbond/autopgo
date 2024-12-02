# This script exports environment variables used by the various components when running in a development environment
# along with minio & nats from the docker compose file.

export AWS_ACCESS_KEY_ID=minio
export AWS_SECRET_ACCESS_KEY=password
export AWS_REGION=dev
export AUTOPGO_BLOB_STORE_URL="s3://default?endpoint=http://localhost:9000&use_path_style=true&disable_https=true"

export NATS_SERVER_URL=localhost:4222
export AUTOPGO_EVENT_WRITER_URL="nats://profile?natsv2=true"
export AUTOPGO_EVENT_READER_URL="nats://profile?natsv2=true&queue=worker"

export AUTOPGO_LOG_LEVEL=debug
export AUTOPGO_API_URL="http://localhost:8080"
export AUTOPGO_APP=autopgo
export AUTOPGO_SAMPLE_SIZE=10
export AUTOPGO_DEBUG=true
