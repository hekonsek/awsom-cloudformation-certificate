# Validated ACM certificate resource for AWS CloudFormation 

## Usage

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
        S3Key: awsom-cloudformation-certificate.zip
      Role: !Sub ${CloudFormationCertificateResourceRole.Arn}
      Timeout: 360
  Certificate:
    Type: Custom::Certificate
    Properties:
          ServiceToken: !Sub ${CloudFormationCertificateResource.Arn}
          Domain: '*.subdomain.example.com'
          HostedZone: example.com
```