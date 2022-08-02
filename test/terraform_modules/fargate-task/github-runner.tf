resource aws_ecs_task_definition oracledb_e2e_runner {
  family = "github-task--oracledb-e2e-tester"

  task_role_arn            = data.terraform_remote_state.base_framework.outputs.github_runner.aws_iam_role.task_role.arn
  execution_role_arn       = data.terraform_remote_state.base_framework.outputs.github_runner.aws_iam_role.execution_role.arn
  requires_compatibilities = ["FARGATE"]

  cpu          = 2 * 1024 # Measured in shares: 1024 shares == 1 vCPU
  memory       = 4 * 1024  # Measured in megabytes
  network_mode = "awsvpc"

  container_definitions = jsonencode([
    {
      name = "provisioner",
      #cpu    = 2 * 1024,  # Measured in shares: 1024 shares == 1 vCPU
      #memory = 4 * 1024,  # Measured in megabytes

      essential              = true,
      readonlyRootFilesystem = false,

      image = "jperezflorez123/nri-oracledb:devel-tooling-latest",

      logConfiguration = {
        "logDriver" = "awslogs"
        "options"   = {
          "awslogs-group" : data.terraform_remote_state.base_framework.outputs.github_runner.aws_cloudwatch_log_group.github_task_runner.name,
          "awslogs-region" : var.aws_region,
          "awslogs-stream-prefix" : split("/", data.terraform_remote_state.base_framework.outputs.github_runner.aws_cloudwatch_log_group.github_task_runner.name)[1]
        }
      }
    }
  ])
}
