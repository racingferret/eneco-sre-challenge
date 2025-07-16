# Eneco SRE challenge README

## 1. Kubernetes Deployment

The following document outlines the process to build and deploy the provided application.  The application and Terraform files have been moved to the `app` and `terraform` folders respectively so as to keep the repository neat and to group like files.  Before reading the rest of the document, the following assumptions were made:

- The docker image was stored on my personal dockerhub account
- THe Helm chart is deployed to a `minikube` installation on my local machine

### 1.1 Building the image

The Dockerfile is uses the `python:alpine3.18` image so as to keep the resultant image size as small as possibe.  The image was built locally using the following command:
```
docker build -t racingferret/eneco-sre-challenge:0.1 .
```

Once built, it was pushed to dockerhub account with the flowing commands:
```
docker login
docker push racingferret/eneco-sre-challenge:0.1
```

### 1.2 Creating the Helm chart

A foundation for the helm chart was created simply using the following command:
```helm create eneco-sre-challenge```

I then modified the `deployment.yaml` file in the templates folder to include the required environment variables expected by the application.  The `vaules.yaml` was modified to update the reference to the correct Docker image (racingferret/eneco-sre-challenge) as well as including default variables for the required environment variables.  The nginx ingress was also enabled as I was using this as the ingress controller in minikube.

### 1.3 Terraform

The included Terraform is quite simple, but is flexible, allowing
- The k8s cluster to be changed, based on the context of your existing `.kube/config` file.
- Allows for the namespace to be change from the `default` namespace if required.
- Environment variables to be overridden in the tfvars file, depending on how/where you'd like to define these settings (values vs Terraform)

The tfvars file is called `terraform.tfvars` so that it is automatically picked up in a plan/apply for simplicity.  The Helm chart that was created previously was then copied into the `terraform/charts` directory so that Terraform could reference it local to the module (again, for simplicity)

### 1.4 Deployment

Deployment of the Helm chart from this point required one more step to create a `dbpassword` secret in Kubernetes (since I strongly believe there is no reason to ever have a password in plain text and especially committed to a git repo)!  To create the secret, I ran the following command:
```
head /dev/urandom | tr -dc A-Za-z0-9 | head -c 32 | xargs -I {} kubectl create secret generic dbpassword --from-literal=password={}
```

Following this step, the deployment was completed with the following Terraform commands:
```
terraform init
terraform plan
terraform apply
```

### 1.5 Testing output

The output of the application can be tested with `curl` using the following command to hit the various endpoints.  In the example below, it shows the `/data` endpoint:
```
curl --resolve "eneco-sre-challenge.local:80:$( minikube ip )" -i http://eneco-sre-challenge.local/data
```

This will produce the following output, based on the Terraform configuration:
```
HTTP/1.1 200 OK
Date: Tue, 15 Jul 2025 21:27:13 GMT
Content-Type: application/json
Content-Length: 163
Connection: keep-alive

{"DB_PASSWORD":"Xwo3eEx3WY6pnIOn8LlVQXzGdH3RD2MF","API_BASE_URL":"http://eneco-sre-challenge.local","LOG_LEVEL":"debug","MAX_CONNECTIONS":"33","ENVIRONMENT":"dev"}
```

### 1.6 Production considerations

I will highlight the major changes that I would make to ensure this application is more "production ready".
- The image itself would be produced in a pipeline, usually as a stage in a larger deployment pipeline.
- Depending on how the development team operate and the cadence of their releases, versioning docker builds might best be done with short checksums.
- Whilst Kubernetes secrets serve a purpose, I would suggest using a more robust secrets manager for sensitive data to ensure fine grained access control and better auditability.
- Before deployment, each container should be scaned for vulnerabilities.
- Ensure applications are successfully deployed with post-deployment checks and also ensuring the app is logging correctly with observability platform of choice.


## 2. CI/CD workflow

The CICD workflow is not functional in it's current state, but does provide a basic framwork and rationale for the requirements provided in section 2 of the document.  Depending on the technology stack and the perferred languages used, these stages can be changes to implement a Terraform run or a deployment of a different kind.  ChatGPT was used for the creation of some of these stages.  The files can be found in the `.github` folder with the following directory structure:
```
.github
  |_workflows
    - ci.yml
    - build.yml
    - deploy.yml
  - dependabot.yml
  - settings.yml
```

The files are described as follows:

`settings.yml:`  This ensure that the `main` branch is protected, requires at least one approver for a PR and ensures the list of stages for linting, testing, security and scans are run.
`dependabot.yml:`  This file enables NPM packages to be scanned daily and our Docker images to be scanned on a weekly basis.
`workflows/ci.yml:`  The stages in this file are executed to to perform linting, ensure code quality, and run secruity scans.
`workflows/build.yml:`  The stages in this file build and push the Docker image to the Github Container Registry, with a tagging system.  It will also run a security scan against the resultant image.
`workflows/deploy.yml:`  Finally, this file performs some integration tests, contains a flexible deployment stage, perform a post-deployment check and rolls back if the checks fail.  Finally, it will notify the monitoring solution of choice (DataDog in this case).


## 3. Log Parser

The log parser is written in golang and (with help from ChatGPT).  I believe fulfils all requirements in the document, although I wasn't entirely clear on the requirements for section d).  The parser has multiple command line options that can be provided to it to allow for your desired filters.  The help output below shows the options available:
```
./main -h

Usage: ./main [options]

Options:
  -file      string   Path to JSON alert file (default: "sample_alerts.json")
  -severity  string   Filter by severity level (e.g., critical, warning, info)
  -service   string   Comma-separated service names to filter by
  -start     string   Start time in RFC3339 format (default: 2000-01-01T00:00:00Z)
  -end       string   End time in RFC3339 format (default: 2100-01-01T00:00:00Z)
  -last      int      Show alerts from the last X minutes (overrides start/end)

Examples:
  Show all critical alerts in the last 10 minutes:
    ./main -severity=critical -last=10

  Show alerts for "payment-service" and "auth-service":
    ./main -service=payment-service,auth-service -severity=warning -last=30
```

As can be seen, the program looks for the file `sample_alerts.json` by default, but other file names can be passed to it.  An example of the output parsing the `sample_alerts2.json` is as follows:
```
$ ./main -service=payment-processor -file=sample_alerts2.json
JSON format OK!

Grouped Alerts by Service (Ordered by Total Priority):

Service: payment-processor
  Component: database, Total Priority: 15
    - ID: ALT-1024 | Severity: warning | Time: 2024-04-28T09:16:02Z | Metric: cpu_usage | Value: 85 | Threshold: 80 | Deviation: 6.25% | Description: Database CPU usage approaching critical threshold
    - ID: ALT-1025 | Severity: critical | Time: 2024-04-28T10:16:02Z | Metric: cpu_usage | Value: 100 | Threshold: 80 | Deviation: 25.00% | Description: Database CPU usage exceeds critical threshold
  Component: api-gateway, Total Priority: 10
    - ID: ALT-1023 | Severity: critical | Time: 2024-04-28T09:15:22Z | Metric: latency | Value: 2300 | Threshold: 1000 | Deviation: 130.00% | Description: API response time exceeded threshold
```