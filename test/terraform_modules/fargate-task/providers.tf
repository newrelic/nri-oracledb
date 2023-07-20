terraform {
  required_providers {
    aws   = {
      source  = "hashicorp/aws"
      version = ">= 3.0"
    }
  }

  backend s3 {
    bucket         = "nr-coreint-terraform-tfstates"
    dynamodb_table = "nr-coreint-terraform-locking"
    key            = "integrations/oracledb/fargate-task.tfstate"
    region         = "us-east-1"
  }
}

# ########################################### #
#  AWS                                        #
# ########################################### #
provider aws {
  default_tags {
    tags = {
      "owning_team" = "COREINT"
      "purpose"     = "development-integration-environments"
      "integration" = "nri-oracledb"
    }
  }
}
