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
    key            = "base-framework/nri-oracledb.tfstate"
    region         = "us-east-1"
    profile        = "base-framework"
  }
}

# ########################################### #
#  AWS                                        #
# ########################################### #
provider aws {
  region  = var.aws_region
  profile = var.aws_profile

  default_tags {
    tags = {
      "owning_team" = "COREINT"
      "purpose"     = "e2e-nightly-automation"
    }
  }
}

# Variables so we can change them using Environment variables.
variable aws_region {
  type    = string
  default = "us-east-1"
}
variable aws_profile {
  type    = string
  default = "coreint"
}
