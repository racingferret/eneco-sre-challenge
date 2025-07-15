# Eneco SRE challenge README

The following document outlines the process to build and deploy the provided application.  The application and Terraform files have been moved to the `app` and `terraform` folders respectively so as to keep the repository neat and to group like files.  Before reading the rest of the document, the following assumptions were made:

- The docker image was stored on my personal dockerhub account
- THe Helm chart is deployed to a `minikube` installation on my local machine

## Building the image

The Dockerfile is uses the `python:alpine3.18` image so as to keep the resultant image size as small as possibe.  The image was built locally using the following command:
```
docker build -t racingferret/eneco-sre-challenge:0.1 .
```

Once built, it was pushed to dockerhub account with the flowing commands:
```
docker login
docker push racingferret/eneco-sre-challenge:0.1
```

## Creating the Helm chart

A foundation for the helm chart was created simply using the following command:
```helm create eneco-sre-challenge```

I then modified the `deployment.yaml` file in the templates folder to include the required environment variables expected by the application.  The `vaules.yaml` was modified to update the reference to the correct Docker image (racingferret/eneco-sre-challenge) as well as including default variables for the required environment variables.  The nginx ingress was also enabled as I was using this as the ingress controller in minikube.

## Terraform

The included Terraform is quite simple, but is flexible, allowing
- The k8s cluster to be changed, based on the context of your existing `.kube/config` file.
- Allows for the namespace to be change from the `default` namespace if required.
- Environment variables to be overridden in the tfvars file, depending on how/where you'd like to define these settings (values vs Terraform)

The tfvars file is called `terraform.tfvars` so that it is automatically picked up in a plan/apply for simplicity.  The Helm chart that was created previously was then copied into the `terraform/charts` directory so that Terraform could reference it local to the module (again, for simplicity)

## Deployment

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

## Testing output

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

## Production considerations

I will highlight the major changes that I would make to ensure this application is more "production ready".
- The image itself would be produced in a pipeline, usually as a stage in a larger deployment pipeline.
- Depending on how the development team operate and the cadence of their releases, versioning docker builds might best be done with short checksums.
- Whilst Kubernetes secrets serve a purpose, I would suggest using a more robust secrets manager for sensitive data to ensure fine grained access control and better auditability.