package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"time"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/route53"
)
import "github.com/aws/aws-sdk-go/aws/session"

func certificateResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, e error) {
	if event.ResourceProperties["Domain"] == nil {
		e = errors.New("'Domain' property is required")
		return
	}
	domain := event.ResourceProperties["Domain"].(string)
	if event.ResourceProperties["HostedZone"] == nil {
		e = errors.New("'HostedZone' property is required")
		return
	}
	hostedZone := event.ResourceProperties["HostedZone"].(string)

	if event.RequestType == cfn.RequestCreate {
		fmt.Println("Received CREATE event.")

		session, err := newSession()
		if err != nil {
			e = err
			return
		}
		acmService := acm.New(session)
		certificateRequestOutput, err := acmService.RequestCertificate(&acm.RequestCertificateInput{
			DomainName:       aws.String(domain),
			ValidationMethod: aws.String("DNS"),
		})
		if err != nil {
			e = err
			return
		}
		physicalResourceID = *certificateRequestOutput.CertificateArn
		data = map[string]interface{}{"CertificateArn": *certificateRequestOutput.CertificateArn}
		fmt.Printf("Generated resource data: %v\n", data)
		fmt.Printf("Created certificate request with ARN: %s\n", *certificateRequestOutput.CertificateArn)

		err = waitUntilCertificateHasValidationOptions(acmService, *certificateRequestOutput.CertificateArn)
		if err != nil {
			e = err
			return
		}

		certificate, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{CertificateArn: certificateRequestOutput.CertificateArn})
		if err != nil {
			e = err
			return
		}
		recordName := certificate.Certificate.DomainValidationOptions[0].ResourceRecord.Name
		recordValue := certificate.Certificate.DomainValidationOptions[0].ResourceRecord.Value

		route53Service := route53.New(session)

		hostedZoneOutput, err := route53Service.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{DNSName: aws.String(hostedZone + ".")})
		if err != nil {
			e = err
			return
		}

		_, err = route53Service.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: hostedZoneOutput.HostedZones[0].Id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action: aws.String("CREATE"),
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name: recordName,
							Type: aws.String("CNAME"),
							ResourceRecords: []*route53.ResourceRecord{
								{Value: recordValue},
							},
							TTL: aws.Int64(5),
						},
					},
				},
			},
		})
		if err != nil {
			e = err
			return
		}

		err = waitUntilCertificateIsValidated(acmService, *certificateRequestOutput.CertificateArn)
		if err != nil {
			e = err
			return
		}
	} else if event.RequestType == cfn.RequestDelete {
		fmt.Println("Received DELETE event.")

		session, err := newSession()
		if err != nil {
			e = err
			return
		}
		acmService := acm.New(session)
		certificates, err := acmService.ListCertificates(&acm.ListCertificatesInput{})
		if err != nil {
			e = err
			return
		}
		for _, cert := range certificates.CertificateSummaryList {
			if *cert.DomainName == domain {
				certificate, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{CertificateArn: cert.CertificateArn})
				if err != nil {
					e = err
					return
				}

				recordName := certificate.Certificate.DomainValidationOptions[0].ResourceRecord.Name
				recordValue := certificate.Certificate.DomainValidationOptions[0].ResourceRecord.Value

				route53Service := route53.New(session)

				hostedZoneOutput, err := route53Service.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{DNSName: aws.String(hostedZone + ".")})
				if err != nil {
					e = err
					return
				}

				_, err = route53Service.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
					HostedZoneId: hostedZoneOutput.HostedZones[0].Id,
					ChangeBatch: &route53.ChangeBatch{
						Changes: []*route53.Change{
							{
								Action: aws.String("DELETE"),
								ResourceRecordSet: &route53.ResourceRecordSet{
									Name: recordName,
									Type: aws.String("CNAME"),
									ResourceRecords: []*route53.ResourceRecord{
										{Value: recordValue},
									},
									TTL: aws.Int64(60),
								},
							},
						},
					},
				})
				if err != nil {
					e = err
					return
				}

				_, err = acmService.DeleteCertificate(&acm.DeleteCertificateInput{CertificateArn: cert.CertificateArn})
				if err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	} else if event.RequestType == cfn.RequestUpdate {
		fmt.Println("Received UPDATE event. Ignoring.")
	}

	return
}

func main() {
	lambda.Start(cfn.LambdaWrap(certificateResource))
}

// newSession returns session which respects:
// - environment variables
// - `~/aws/.config` and `~/aws/.credentials` files
//
// Example:
//
//     import "github.com/hekonsek/awsom-session"
//     ...
//     err, sess := awsom_session.newSession()
func newSession() (*session.Session, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// waitUntilCertificateHasValidationOptions asserts that certificate request has validation option assigned to it.
// This is needed because validation option is assigned to request after a small delay.
func waitUntilCertificateHasValidationOptions(acmService *acm.ACM, certificateArn string) error {
	fmt.Printf("Ensuring that certificate with ARN %s has validation option assigned to it.\n", certificateArn)
	for i := 0; i < 10; i++ {
		hasCertificate, err := certificateHasValidationOptions(acmService, certificateArn)
		if err != nil {
			return err
		}
		if hasCertificate {
			return nil
		}
		fmt.Printf("Cannot find validation options for certificate with ARN %s. Retrying in 6 seconds...\n", certificateArn)
		time.Sleep(6 * time.Second)
	}
	return errors.New("no validation option for certificate - timed out after a minute")
}

func certificateHasValidationOptions(acmService *acm.ACM, certificateArn string) (bool, error) {
	certificate, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{CertificateArn: aws.String(certificateArn)})
	if err != nil {
		return false, err
	}
	return len(certificate.Certificate.DomainValidationOptions) > 0, nil
}

func waitUntilCertificateIsValidated(acmService *acm.ACM, certificateArn string) error {
	fmt.Printf("Ensuring that certificate with ARN %s is validated.\n", certificateArn)
	for i := 0; i < 150; i++ {
		isValidated, err := certificateIsValidated(acmService, certificateArn)
		if err != nil {
			return err
		}
		if isValidated {
			return nil
		}
		fmt.Printf("Certificate with ARN %s is not validated yet. Retrying in 6 seconds...\n", certificateArn)
		time.Sleep(6 * time.Second)
	}
	return errors.New("certificate validation timed out after 15 minutes")
}

func certificateIsValidated(acmService *acm.ACM, certificateArn string) (bool, error) {
	certificate, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{CertificateArn: aws.String(certificateArn)})
	if err != nil {
		return false, err
	}
	return *certificate.Certificate.Status == acm.CertificateStatusIssued, nil
}
