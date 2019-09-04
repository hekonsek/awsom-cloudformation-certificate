all: build

build:
	GO111MODULE=on go build awsom-cloudformation-certificate.go
	zip awsom-cloudformation-certificate.zip awsom-cloudformation-certificate

deploy: build
	aws s3 cp awsom-cloudformation-certificate.zip s3://capsilon-awsom/awsom-cloudformation-certificate.zip --acl=public-read