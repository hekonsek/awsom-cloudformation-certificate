# Validated ACM certificate resource for AWS CloudFormation 

This is custom CloudFormation resource for validated [ACM](https://aws.amazon.com/certificate-manager) HTTPS/CA certificate. It
creates ACM request together with DNS CNAME recordset in Route53 for validation purposes. The resource creation process
will not be completed until certificate is not validated. It means that when you define ACM certificate like this...

```
Certificate:
  Type: Custom::Certificate
  Properties:
    ServiceToken: !Sub ${CloudFormationCertificateResource.Arn}
    Domain: '*.subdomain.example.com'
    HostedZone: example.com
``` 

...you can be sure that successfully provisioned stack included properly validated, read to  use, ACM certificate.  

## Usage

The latest release of this resource can be found [here](s3://capsilon-awsom/awsom-cloudformation-certificate-0.2.0.zip).
I can't promise this hosting site will be available in the future, so I highly recommend to download the zip file and
host it in your own S3 bucket.

In order to use this resource you have to define its definition as a Lambda in the first place. This is standard 
practice for custom CloudFormation resources.

```
Resources:
  CloudFormationCertificateResourceRole:
    Type: AWS::IAM::Role
    Properties: 
      AssumeRolePolicyDocument: 
        Version: '2012-10-17'
        Statement: 
        - Effect: Allow
          Principal: 
            Service: lambda.amazonaws.com
          Action: 
          - sts:AssumeRole
      Path: '/'
      Policies: 
            - PolicyName: logs
              PolicyDocument: 
                Statement: 
                - Effect: Allow
                  Action: 
                  - logs:CreateLogGroup
                  - logs:CreateLogStream
                  - logs:PutLogEvents
                  Resource: arn:aws:logs:*:*:*
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/AWSCertificateManagerFullAccess
        - arn:aws:iam::aws:policy/AmazonRoute53FullAccess
  CloudFormationCertificateResource:
    Type: AWS::Lambda::Function
    Properties:
      FunctionName: cloudformation-certificate-resource
      Runtime: go1.x
      Handler: awsom-cloudformation-certificate 
      Code: 
        S3Bucket: capsilon-awsom
        S3Key: awsom-cloudformation-certificate-0.2.0.zip
      Role: !Sub ${CloudFormationCertificateResourceRole.Arn}
      Timeout: 1200

  Certificate:
    Type: Custom::Certificate
    Properties:
          ServiceToken: !Sub ${CloudFormationCertificateResource.Arn}
          Domain: '*.subdomain.example.com'
          HostedZone: example.com
```