# `semblame`
LLM-powered git history analysis

## Overview

`semblame` is a developer tool that enhances the traditional `git blame` by using semantic embeddings and LLMs to provide insights into why code changes were made, not just who made them.

## Features

- Extract Git commit metadata and diffs
- Generate embeddings for commit messages and code changes
- Store embeddings in a vector database for semantic search
- Use Retrieval-Augmented Generation (RAG) with LLMs for natural language queries

## CLI Usage

### ingest

Walk the Git history, generate embeddings, and store them in the database.

```bash
./semblame ingest [path/to/repo]
```

- `path/to/repo`: Optional. The path to the Git repository (defaults to the current directory).

### query

Query the indexed history with a natural language question.

```bash
./semblame query [path/to/repo] "Your question here"
```

- `path/to/repo`: The path to the Git repository.
- `"Your question here"`: The natural language query to ask.