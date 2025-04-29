Here’s a self-contained project description document tailored for sharing in
Cursor, designed to establish clear context, objectives, and implementation
scope for your initial development phase of **Semblame**:

---

# **Semblame**

**Tagline**: *Semantic Git blame: understanding not just who changed a line of
*code, but why.*

## **Project Overview**

Semblame is a developer tool that enhances the traditional `git blame` by
incorporating semantic reasoning over version history. It uses Git commit data
and code diffs to generate embeddings and support natural language querying via
LLMs. The goal is to help developers understand the *rationale* behind code
changes, not just track *who* made them and *when*.

## **Initial Focus (Phase 1)**

This phase focuses on extracting useful embeddings from Git history — commit
messages and code diffs — and enabling semantic search over them. By combining
these embeddings with a Retrieval-Augmented Generation (RAG) setup, Semblame
will support queries like:

- *"Why was this validation added?"*
- *"When did this logic change, and what motivated it?"*

No complex lineage tracking or file diffs across multiple commits yet — just a
powerful baseline to build on.

---

## **Key Phase 1 Goals**

- Parse Git history to extract:
  - Commit metadata (message, author, date)
  - File-level diffs per commit
- Embed commit messages and code changes using a code-aware embedding model
- Store embeddings in a vector database
- Implement a CLI tool to run semantic queries against the index using LLM + RAG

---

## **Architecture Overview**

### Git Extraction
- Use `go-git` (or shell out to `git`) to walk commit history
- For each commit:
  - Capture the message and metadata
  - Extract unified diff per file

### Embedding Pipeline
- Preprocess each data item (e.g. commit message + diff) into plain text chunks
- Generate embeddings using a suitable model:
  - `text-embedding-3-small` (OpenAI)
  - `bge-code` (open-source alternative)
- Store with metadata in a vector store (e.g. Qdrant, SQLite+FAISS)

### RAG Query Interface
- User provides a natural language query (e.g., “why did we add this error
  check?”)
- The system:
  1. Embeds the query
  2. Retrieves top-K relevant commit/diff chunks from the vector store
  3. Feeds them to an LLM for a synthesized response

### CLI Commands
- `semblame ingest`: walk repo history, embed relevant data, and index it
- `semblame query "<question>"`: retrieve relevant history and produce a summary

---

## **Advantages of This Approach**

- Quick initial value without complex diff tracking logic
- LLMs can surface surprising connections early on
- Modular foundation: data ingestion, embedding, storage, and querying are
  decoupled

---

## **Future Extensions (Beyond Phase 1)**

- Track code lineage and symbol/function evolution over time
- Line-specific blame with change rationales
- Fine-tune summarization models on commit-diff pairs
- IDE integrations (e.g., Cursor, VS Code)
- Visualization of semantic evolution