all: build

VERSION=0.3.0

build:
	GO111MODULE=on go build awsom-cloudformation-certificate.go
	zip awsom-cloudformation-certificate-$(VERSION).zip awsom-cloudformation-certificate

deploy: build
	aws s3 cp awsom-cloudformation-certificate-$(VERSION).zip s3://capsilon-awsom/awsom-cloudformation-certificate-$(VERSION).zip --acl=public-read

version:
	sed -i "s/${VERSION}/${NEW_VERSION}/g" readme.md Makefile
