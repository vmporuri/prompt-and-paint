# Prompt and Paint

[![GoDoc](https://godoc.org/github.com/vmporuri/prompt-and-paint?status.svg)](https://godoc.org/github.com/vmporuri/prompt-and-paint)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmporuri/prompt-and-paint)](https://goreportcard.com/report/github.com/vmporuri/prompt-and-paint)

A fun new party game: Prompt and Paint!

## Installation

Run the app locally with Docker.

1. Add your OpenAI API key to api-key.env in the top-level project directory.

```bash
OPENAI_API_KEY="$YOUR_API_KEY"
```

2. Build the Docker image.

```bash
docker build -t vmporuri/prompt-and-paint .
```

3. Start the application with Docker Compose.

```bash
docker compose up
```

4. Visit URL in browser.

```
http://localhost:8080
```
