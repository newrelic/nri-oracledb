resource aws_db_subnet_group oracle_db {
  name       = "oracle-db--${terraform.workspace}"
  tags = {
    from_github = length(regexall("ghrun-", terraform.workspace)) > 0 ? "yes" : "no"
    workspace = terraform.workspace
  }

  subnet_ids = [
    # If I use a random provider random numbers could collide so https://xkcd.com/221/
    data.terraform_remote_state.base_framework.outputs.common_networking.aws_subnet.private_subnets[0].id,
    data.terraform_remote_state.base_framework.outputs.common_networking.aws_subnet.private_subnets[1].id,
    data.terraform_remote_state.base_framework.outputs.common_networking.aws_subnet.private_subnets[5].id,
  ]
}

resource aws_db_instance oracle_db {
  tags = {
    Name = "oracle-db--${terraform.workspace}"
    from_github = length(regexall("ghrun-", terraform.workspace)) > 0 ? "yes" : "no"
    workspace = terraform.workspace
  }

  # https://docs.aws.amazon.com/AmazonRDS/latest/OracleReleaseNotes/Welcome.html
  engine         = "oracle-ee"
  engine_version = "19.0.0.0.ru-2022-01.rur-2022-01.r1"
  # engine_version = "21.0.0.0.ru-2022-01.rur-2022-01.r1"

  instance_class       = "db.t3.small"
  db_subnet_group_name = aws_db_subnet_group.oracle_db.name

  allocated_storage          = 20
  auto_minor_version_upgrade = true
  backup_retention_period    = 0
  skip_final_snapshot = true

  # The Oracle System ID (SID) of the created DB instance. If you specify null, the default value ORCL is used.
  # You can't specify the string NULL, or any other reserved word, for DBName.
  # Default: ORCL
  # Constraints: Can't be longer than 8 characters
  db_name  = "ORACLE"
  username = "foo"
  password = "foobarbaz"  # HARDCODED. Change to use lastpass.
}
