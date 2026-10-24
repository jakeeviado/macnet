## Introduction

**MACnet - Multi-language Analysis & Code Network** is a command-line tool used for multi-language code analysis. It flags risks and bugs to assist manual code reviews, particularly on AI-assisted codebases.

MACnet Supports:

- **Languages:** Python, Java, JS/TS, Go, Rust, C/C++, Ruby, PHP, C#, Kotlin, and Swift.
- **Analysis Engine:** Uses both heuristic checks and an ONNX machine learning model.
- **Dataset Management:** Runs offline; includes tools to collect and label local results to retrain models. [See Machine Learning Documentation](model)
- **Pipeline & Workflow Support:** Integrates with GitHub Actions, GitLab CI, and Jenkins. Features Git diff scanning and build gating.
- **Configuration:** Managed via .macnet.yaml or CLI flags. [See Sample YAML file](.macnet.example.yaml)

---

MACnet parses source code and computes a normalized vector (a list of numbers representing different code traits, scaled from 0.0 to 1.0) across various Risk Signals via pkg/analyzer/signals/. These extracted feature weights are passed directly to an ONNX (an open format used to run machine learning models across different backends) Random Forest model (model.onnx) to classify code quality and detect AI-generated vs. human-written patterns.

**This currrent active heuristic signals are subject to adjustment and expansion over time.**

| Signal                   | Description                                      |
| :----------------------- | :----------------------------------------------- |
| `comment_density`        | Comment-to-code ratio.                           |
| `generic_naming`         | Use of placeholder names (e.g., `temp`, `data`). |
| `repetitive_patterns`    | File-level code duplication.                     |
| `code_complexity`        | Control flow density.                            |
| `formatting_consistency` | Indentation/style uniformity.                    |
| `comment_redundancy`     | Comments that mirror code logic.                 |
| `emoji_sentiment`        | Presence of non-standard emojis.                 |
| `identifier_order`       | Suspiciously perfect alphabetical ordering.      |
| `defensive_ratio`        | Ratio of error handling to functional logic.     |

---

### In Action

MACnet scans and detects.

```
Scanning path: ./examples
Threshold: 0.70
Found 5 files to scan

================================================================================
                       ⌀ MACnet - Code Analysis Report
================================================================================

Detection Mode:       Heuristics Only
Total Files Scanned:  5
Files Detected:       2
Average Score:        0.54
Detection Threshold:  0.70

[!] HIGH-RISK FILES (above threshold):
--------------------------------------------------------------------------------
examples\flagged\python\flagged_2.py
   Score: 0.80 | Language: python
   Signals:
     • emoji_sentiment (0.80): High density of informal emojis in production code
       Emojis found: 🚀 (1), ✨ (1), ✅ (1)
     • comment_density (0.50): High comment density (>20%)
       Comment lines: 5 / 22 (22.7%)
     • formatting_consistency (0.50): Unusually rigid indentation uniformity across a large file
       Unique indentation levels: 3

[?] REVIEW RECOMMENDED (below threshold but noteworthy):
--------------------------------------------------------------------------------
examples\flagged\python\flagged_3.py
   Score: 0.60 | Language: python
   Signals:
     • identifier_order (0.60): Large identifier blocks in perfect alphabetical order warrant a closer look
       Perfectly sorted blocks (6+ items): 1
     • formatting_consistency (0.50): Unusually rigid indentation uniformity across a large file
       Unique indentation levels: 2
     • emoji_sentiment (0.30): Informal emojis detected in comments or log messages
       Emojis found: ✨ (1), 🤖 (1)

================================================================================
FAILED: 1 file(s) require attention (risk threshold exceeded)
Note: Scores represent heuristic and/or ML-based risk estimates. Review is recommended for flagged files.
================================================================================
```

**Common Usage Examples or Workflows & Combinations:**

```bash
./macnet

# Scan current directory
./macnet scan --path .

# Scan with specific threshold
./macnet scan --path ./src --threshold 0.8

# Scan specific language/s
./macnet scan --path . --languages python,javascript

# Output as JSON
./macnet scan --path . --json

# Fail build if high-risk patterns are detected above threshold
./macnet scan --path . --fail-on-detection --threshold 0.75

# Collect training data for Machine Learning Model (directory should be ready inside the project)
./macnet scan --path ./examples/flagged --collect --label flagged
./macnet scan --path ./examples/human --collect --label human

# Hybrid Mode (Default when --model is provided)
# Blends ML probability with heuristic patterns using a weighted average.
./macnet scan --path . --model ./model/model.onnx --ml-weight 0.7 --verbose

# Heuristics Only
# Ignores the ML model entirely and looks for raw code patterns.
./macnet scan --path . --no-ml --verbose

# ML Only
# Forces the detector to rely primarily on the model. 
./macnet scan --path . --ml-only --model ./model/model.onnx --verbose
```

**Flags:**
- `--path, -p` - Path to scan (default: current directory)
- `--languages, -l` - Comma-separated list of languages to scan
- `--threshold, -t` - Detection threshold 0.0-1.0 (default: 0.7)
- `--fail-on-detection` - Exit with error code if a risky code is detected
- `--exclude` - Paths to exclude from scanning
- `--verbose` - Verbose output with detailed signals
- `--json` - Output results in JSON format
- `--collect` - Save scan results to local training CSV
- `--label` - Label for training data: 'flagged' or 'human'
- `--model` - Path to ONNX model file
- `--no-ml` - Disable ML inference (heuristics only)
- `--ml-only` - Use ML only (fail if model not available)
- `--ml-weight` - Weight given to ML score (0.0-1.0)
- `--git-diff` - Scan only files changed against the specified branch

---

### The easy way to use this

CI/CD Integration (Github Actions, GitLab CI, and Jenkins)

**GitHub Actions**

```yaml
name: MACnet
on: [pull_request]

jobs:
  macnet:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      
      - name: Download MACnet
        run: |
          curl -L https://github.com/jakeeviado/macnet/releases/latest/download/macnet-linux-amd64 -o macnet
          chmod +x macnet
      
      - name: Scan Code
        run: ./macnet scan --path . --threshold 0.75 --fail-on-detection
```

**GitLab CI**

```yaml
macnet:
  stage: test
  script:
    - curl -L https://github.com/jakeeviado/macnet/releases/latest/download/macnet-linux-amd64 -o macnet
    - chmod +x macnet
    - ./macnet scan --path . --threshold 0.75 --fail-on-detection
  only:
    - merge_requests
```

**Jenkins**

```groovy
pipeline {
    agent any
    stages {
        stage('MACnet') {
            steps {
                sh '''
                    curl -L https://github.com/jakeeviado/macnet/releases/latest/download/macnet-linux-amd64 -o macnet
                    chmod +x macnet
                    ./macnet scan --path . --threshold 0.75 --fail-on-detection
                '''
            }
        }
    }
}
```

**Option for Git Diff Scanning (for CI/CD)**

```bash
# Scan only files changed in PR
./macnet scan --git-diff origin/main --threshold 0.75
```

**Local Installation**

You can also download the latest pre-compiled binaries from the [Releases](https://github.com/jakeeviado/macnet/releases) page and install them directly to your machine (supported on Linux & Windows), or clone the project and build it from source:

```bash
git clone https://github.com/jakeeviado/macnet.git
cd macnet
go build
```

---

### Disclaimer
MACnet is an aid for code quality and review; it is not intended to determine authorship or replace human judgment. Results are probabilistic and may include false positives.

This is a hobby and personal project

### License
Open Sourced - You are free to fork, modify, distribute, and train your own ONNX model on this.

Licensed under the MIT License - see [LICENSE](LICENSE) file for details.
