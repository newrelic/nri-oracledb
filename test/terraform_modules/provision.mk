TERRAFORM_FILES     := $(CURDIR)/test/terraform_modules
TERRAFORM_MODULE    ?= aws_managed_instance
TERRAFORM_WORKSPACE ?= $(shell whoami)
TERRAFORM_CHDIR     := -chdir="$(TERRAFORM_FILES)/$(TERRAFORM_MODULE)"

AWS_PROFILE ?= coreint

DOCKER_USER  := $(shell id -u)
DOCKER_GROUP := $(shell id -g)

provision: terraform-init terraform-workspace-new terraform-apply
deprovision: terraform-init terraform-workspace-new terraform-destroy

.PHONY: terraform-init
terraform-init:
	@echo "=== $(INTEGRATION) === [ terraform ]: Initializing terraform..."
	@TF_IN_AUTOMATION=1 \
	 terraform $(TERRAFORM_CHDIR) init -input=false
#	@docker run --rm -ti --platform linux/amd64 -u $(DOCKER_USER):$(DOCKER_GROUP) --pull always \
#                -v $${HOME}/.aws:/.aws \
#                -v "$(TERRAFORM_FILES)/$(TERRAFORM_MODULE):$(TERRAFORM_FILES)/$(TERRAFORM_MODULE)" \
#                -w "$(TERRAFORM_FILES)/$(TERRAFORM_MODULE)" \
#                -e AWS_ACCESS_KEY_ID=$${AWS_ACCESS_KEY_ID} \
#                -e AWS_SECRET_ACCESS_KEY=$${AWS_SECRET_ACCESS_KEY} \
#                -e AWS_DEFAULT_REGION=$${AWS_DEFAULT_REGION} \
#                -e AWS_PROFILE=$${AWS_PROFILE} \
#                -e TF_VAR_aws_profile=$${AWS_PROFILE} \
#                -e TF_WORKSPACE=$(TERRAFORM_WORKSPACE) \
#                -e TF_IN_AUTOMATION=1 \
#                hashicorp/terraform \
#                init $(TERRAFORM_ARGS)

.PHONY: terraform-workspace-new
terraform-workspace-new:
	@echo "=== $(INTEGRATION) === [ terraform ]: Creating terraform workspace..."
	@TF_IN_AUTOMATION=1 \
	 terraform $(TERRAFORM_CHDIR) workspace new $(TERRAFORM_WORKSPACE) || \
	 terraform $(TERRAFORM_CHDIR) workspace select $(TERRAFORM_WORKSPACE)

.PHONY: terraform-apply
terraform-apply:
	@echo "=== $(INTEGRATION) === [ terraform ]: Applying terraform..."
	@TF_WORKSPACE=$(TERRAFORM_WORKSPACE) \
	 TF_IN_AUTOMATION=1 \
	 terraform $(TERRAFORM_CHDIR) apply -input=false -auto-approve

.PHONY: terraform-destroy
terraform-destroy:
	@echo "=== $(INTEGRATION) === [ terraform ]: Destroying terraform..."
	@TF_WORKSPACE=$(TERRAFORM_WORKSPACE) \
	 TF_IN_AUTOMATION=1 \
	 terraform $(TERRAFORM_CHDIR) destroy -input=false -auto-approve
