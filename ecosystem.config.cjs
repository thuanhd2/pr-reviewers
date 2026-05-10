module.exports = {
  apps: [
    {
      name: 'pr-reviewer-api',
      script: './bin/api',
      env: {
        PORT: '8080',
        DATABASE_URL: 'postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable',
        REDIS_URL: 'localhost:6379',
        CONFIG_PATH: './config.yaml',
      },
      log_file: './logs/api.log',
      error_file: './logs/api-error.log',
      max_restarts: 5,
      restart_delay: 5000,
    },
    {
      name: 'pr-reviewer-worker',
      script: './bin/worker',
      env: {
        DATABASE_URL: 'postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable',
        REDIS_URL: 'localhost:6379',
        CONFIG_PATH: './config.yaml',
      },
      log_file: './logs/worker.log',
      error_file: './logs/worker-error.log',
      max_restarts: 5,
      restart_delay: 10000,
    },
  ],
}
