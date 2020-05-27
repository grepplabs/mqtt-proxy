.DEFAULT_GOAL := info
.PHONY: info

SSO_PROFILE=default

info:
	@echo "################################################################################"
	@echo "### Usage:"
	@echo "###    make <target>"
	@echo "################################################################################"


.PHONY: tf-init
tf-init:
	terraform init -input=false

.PHONY: tf-plan
tf-plan: tf-init
	terraform plan -input=false

.PHONY: tf-apply
tf-apply: tf-init
	terraform apply -input=false -auto-approve

.PHONY: tf-destroy
tf-destroy:
	terraform destroy -input=false -auto-approve

.PHONY: tf-output
tf-output:
	terraform output

.PHONY: sso-login
sso-login:
	aws sso login --profile $(SSO_PROFILE)
	# install tool from https://github.com/grepplabs/aws-sso/releases
	@echo "Store temporary credentials in the ~/.aws/credentials"
	aws-sso credentials refresh --profile $(SSO_PROFILE)
