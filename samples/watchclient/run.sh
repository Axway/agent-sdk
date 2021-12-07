export WC_TENANT_ID=426937327920148
export WC_HOST=localhost
export WC_PORT=8080
export WC_TOPIC_SELF_LINK=/management/v1alpha1/watchtopics/wc-env-discoveryagents
export WC_AUTH_PRIVATE_KEY=/Users/tjohnson/Desktop/private_key.pem
export WC_AUTH_PUBLIC_KEY=/Users/tjohnson/Desktop/public_key.pem
export WC_AUTH_CLIENT_ID=watch-service_51e15b63-0050-45e5-92e9-ac4fa7d116db
export WC_INSECURE=true
export WC_LOG_LEVEL=debug
export WC_LOG_FORMAT=line

go run ./main.go
