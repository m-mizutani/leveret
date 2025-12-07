# Embedding Example

This example demonstrates how to generate text embeddings using the Gemini API and calculate cosine distances between texts.

## Features

- Generate embeddings for multiple texts
- Calculate pairwise cosine distances between all input texts
- Support for custom embedding dimensions
- Human-readable console output (default) or JSON format
- Optional vector display

## Prerequisites

- Google Cloud project with Gemini API enabled
- Application Default Credentials configured: `gcloud auth application-default login`

## Usage

### Basic Usage

```bash
go run examples/embedding/main.go --gemini-project your-project "cat" "dog" "car"
```

Output:
```
=== Embeddings ===
Text: cat
Text: dog
Text: car

=== Cosine Distances ===
cat <-> dog: 0.1523
cat <-> car: 0.8234
dog <-> car: 0.7891
```

### Show Embedding Vectors

Use `-v` or `--show-vector` to display the embedding vectors:

```bash
go run examples/embedding/main.go -v --gemini-project your-project "cat" "dog"
```

Output:
```
=== Embeddings ===
Text: cat
  Vector: [0.123 0.456 0.789 ...]
Text: dog
  Vector: [0.234 0.567 0.890 ...]

=== Cosine Distances ===
cat <-> dog: 0.1523
```

### JSON Output

Use `-j` or `--json` for JSON format output:

```bash
go run examples/embedding/main.go -j --gemini-project your-project "cat" "dog"
```

Output:
```json
{
  "embeddings": [
    {
      "text": "cat"
    },
    {
      "text": "dog"
    }
  ],
  "distances": [
    {
      "text1": "cat",
      "text2": "dog",
      "distance": 0.1523
    }
  ]
}
```

### Custom Dimensions

Use `-d` or `--dimensions` to specify embedding dimensions (default: 8):

```bash
go run examples/embedding/main.go -d 256 --gemini-project your-project "cat" "dog"
```

## Command-Line Options

| Flag | Alias | Environment Variable | Default | Description |
|------|-------|---------------------|---------|-------------|
| `--dimensions` | `-d` | `LEVERET_EMBEDDING_DIMENSIONS` | 8 | Embedding dimensions |
| `--gemini-project` | - | `LEVERET_GEMINI_PROJECT` | - | Google Cloud project ID (required) |
| `--gemini-location` | - | `LEVERET_GEMINI_LOCATION` | us-central1 | Google Cloud location |
| `--show-vector` | `-v` | `LEVERET_SHOW_VECTOR` | false | Show embedding vectors |
| `--json` | `-j` | `LEVERET_OUTPUT_JSON` | false | Output in JSON format |

## Using Environment Variables

You can set environment variables instead of passing flags:

```bash
export LEVERET_GEMINI_PROJECT=your-project
export LEVERET_EMBEDDING_DIMENSIONS=256
go run examples/embedding/main.go "cat" "dog" "car"
```

## Understanding Cosine Distance

Cosine distance is calculated as `1 - cosine_similarity`, where:
- **0.0**: Identical vectors (most similar)
- **1.0**: Orthogonal vectors (least similar)
- **2.0**: Opposite vectors

Lower values indicate higher semantic similarity between texts.

## Example Use Cases

### Compare Related Words

```bash
go run examples/embedding/main.go --gemini-project your-project \
  "cat" "kitten" "dog" "puppy" "car" "automobile"
```

You'll see that:
- "cat" and "kitten" have low distance (semantically similar)
- "dog" and "puppy" have low distance (semantically similar)
- "car" and "automobile" have low distance (synonyms)
- "cat" and "car" have high distance (semantically different)

### Compare Sentences

```bash
go run examples/embedding/main.go --gemini-project your-project \
  "I love programming" \
  "Coding is my passion" \
  "The weather is nice today"
```

The first two sentences will have lower distance than either has with the third.

### Analyze Security Alerts (with higher dimensions)

```bash
go run examples/embedding/main.go -d 768 --gemini-project your-project \
  "Suspicious login from unusual IP address" \
  "Failed authentication attempt detected" \
  "S3 bucket made public" \
  "Unauthorized database access"
```

This can help identify similar security events for alert deduplication or clustering.

## Building

```bash
go build -o embedding ./examples/embedding
./embedding --gemini-project your-project "text1" "text2"
```
