## Introduction

**MACnet** (Multi-language Analysis & Code Network) is a command-line tool used for multi-language code analysis. It flags risks and bugs to assist manual code reviews, particularly on AI-assisted codebases.

MACnet Supports:

- **Languages:** Python, Java, JS/TS, Go, Rust, C/C++, Ruby, PHP, C#, Kotlin, and Swift.
- **Analysis Engine:** Uses both heuristic checks and an ONNX machine learning model.
- **Detection Profiles:** `loose` for everyday code review, `strict` for assessments and other settings that need deeper evidence. [See Profiles](#profiles)
- **Dataset Management:** Runs offline; includes tools to collect and label local results to retrain models. [See Machine Learning Documentation](model)
- **Pipeline & Workflow Support:** Integrates with GitHub Actions, GitLab CI, and Jenkins. Features Git diff scanning and build gating.
- **Configuration:** Managed via .macnet.yaml or CLI flags. [See Sample YAML file](.macnet.example.yaml)

---

MACnet parses source code and computes a normalized vector (a list of numbers representing different code traits, scaled from 0.0 to 1.0) across various Risk Signals via pkg/analyzer/signals/. These extracted feature weights are passed directly to an ONNX (an open format used to run machine learning models across different backends) Random Forest model (model.onnx) to classify code quality and detect AI-generated vs. human-written patterns.

**The currently active heuristic signals are subject to fine-tuning and expansion over time.**

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
> ./macnet scan --path ./examples --profile strict --verbose
Scanning path: ./examples
Profile: strict
Threshold: 0.70
Found 7 files to scan

================================================================================

                           ░▄█ ▀▀▄ ▀▀█  ▀▀▄ ▀▀█  ▀▀▀
                           █ ░ ░ ▓ ░    █ ▓ ░     ░
                           ▓ ▒ ▒▀▒ ▒    ▓ ▒ ▒▀    ▒
                           ▒ ▓ ▓ ░ ▓ ▄  ▒ ░ ▓ ▄   ▓
                           ▀ ▀ ▀ ▀ ▀▀▀  ░ ▀ ▀▀█   ░
                                   ▀▀▀   ▀▀▀ ▀▀▀
                            MACnet Analysis Report

================================================================================

Detection Mode:       Heuristics Only
Total Files Scanned:  7
Files Detected:       3
Average Score:        0.50
Detection Threshold:  0.70

[!] HIGH-RISK FILES (above threshold):
--------------------------------------------------------------------------------
examples/flagged/python/flagged_1.py
   Score: 0.90 | Language: python
   Signals:
     • generic_naming (0.90)
       Very high use of generic variable names
       generic_count: 47, generic_density: 0.64

     • formatting_consistency (0.50)
       Unusually rigid indentation uniformity across a large file
       non_blank_lines: 52, unique_indent_levels: 3

     • code_complexity (0.30)
       Low control flow density
       control_flow_density: 0.07, control_flow_lines: 5

examples/flagged/python/flagged_2.py
   Score: 0.80 | Language: python
   Signals:
     • emoji_sentiment (0.80)
       High density of informal emojis in production code. Emojis found: 🚀 (1), ✨ (1), ✅ (1)
       emoji_distinct: 3, emoji_total: 3

     • comment_density (0.50)
       High comment density (>20%)
       comment_density: 0.23, comment_lines: 5, total_lines: 22

   Not evaluated: formatting_consistency (file too short to judge indentation)

examples/flagged/python/flagged_5.py
   Score: 0.80 | Language: python
   Signals:
     • comment_density (0.80)
       Excessive comment density detected (>30%)
       comment_density: 0.35, comment_lines: 6, total_lines: 17

     • generic_naming (0.60)
       High use of generic variable names
       generic_count: 8, generic_density: 0.47

   Not evaluated: formatting_consistency (file too short to judge indentation)


[?] REVIEW RECOMMENDED (below threshold but noteworthy):
--------------------------------------------------------------------------------
examples/flagged/python/flagged_3.py
   Score: 0.60 | Language: python
   Signals:
     • identifier_order (0.60)
       line 3: Perfectly sorted identifier block
       block_size: 8

     • emoji_sentiment (0.30)
       Informal emojis detected in comments or log messages. Emojis found: ✨ (1), 🤖 (1)
       emoji_distinct: 2, emoji_total: 2

   Not evaluated: formatting_consistency (file too short to judge indentation)


================================================================================
FAILED: 3 file(s) require attention (risk threshold exceeded)
Note: Scores represent heuristic and/or ML-based risk estimates.
Review is recommended for flagged files.
================================================================================
```

A signal listed under `Not evaluated` could not judge its input (for example, a file that is too short). That is different from a signal that ran and found nothing.

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

# Strict profile (adds repo-baseline and history signals)
./macnet scan --path . --profile strict --verbose

# Scan only what this branch changed, strict profile
./macnet scan --git-diff origin/main --profile strict --verbose

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
- `--profile` - Detection profile: `loose` (default) or `strict`
- `--threshold, -t` - Detection threshold 0.0-1.0 (default: 0.7, or the default of the selected profile)
- `--fail-on-detection` - Exit with error code if a risky code is detected
- `--exclude` - Paths or glob patterns to exclude from scanning (for example `vendor`, `*.pb.go`, `src/gen`)
- `--verbose` - Verbose output with detailed signals
- `--json` - Output results in JSON format
- `--collect` - Save scan results to local training CSV (not supported together with `--git-diff`)
- `--label` - Label for training data: 'flagged' or 'human'
- `--model` - Path to ONNX model file
- `--no-ml` - Disable ML inference (heuristics only)
- `--ml-only` - Use ML only (requires `--model`; fails if the model cannot be loaded)
- `--ml-weight` - Weight given to ML score (0.0-1.0)
- `--git-diff` - Scan only files changed since the merge-base with the specified branch (committed changes, `<branch>...HEAD`)

`--no-ml` and `--ml-only` cannot be combined.

---

### Profiles

| Profile  | Signals that run                                                                   | Typical use                                                      |
| :------- | :--------------------------------------------------------------------------------- | :--------------------------------------------------------------- |
| `loose`  | File-level signals and cheap diff signals                                          | Everyday code review, CI on pull requests                        |
| `strict` | Everything in `loose`, plus repo-baseline, git-history and author-baseline signals | Assessments and other higher-scrutiny review (academic or other) |

Notes:

- `strict` needs full git history. In CI, check out with `fetch-depth: 0` (GitHub Actions), `GIT_DEPTH: "0"` (GitLab CI), or fetch the full history in Jenkins. On a shallow clone MACnet warns and skips the history-based signals.
- Diff-level signals are **advisory**. They are shown in the report as evidence for a reviewer, and they never fail a build, even in `strict`.
- Signals that depend on a repo or author baseline are planned in stages; [HEURISTICS.md](HEURISTICS.md) tracks which are active.

---

### Configuration

Settings can live in a `.macnet.yaml` file in the project root. A flag on the command line always wins over the file, and the file wins over the built-in default. See [.macnet.example.yaml](.macnet.example.yaml) for every option.

```yaml
profile: strict
threshold: 0.75
exclude:
    - vendor
    - "*_test.go"
fail_on_detection: true
```

| YAML key            | Flag                  |
| :------------------ | :-------------------- |
| `profile`           | `--profile`           |
| `threshold`         | `--threshold`         |
| `languages`         | `--languages`         |
| `exclude`           | `--exclude`           |
| `fail_on_detection` | `--fail-on-detection` |
| `verbose`           | `--verbose`           |
| `json`              | `--json`              |
| `model`             | `--model`             |
| `ml_weight`         | `--ml-weight`         |
| `no_ml`             | `--no-ml`             |
| `ml_only`           | `--ml-only`           |

Flags that describe a single run (`--path`, `--git-diff`, `--collect`, `--label`) are flag-only.

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
  variables:
    GIT_DEPTH: "0"   # full history, needed for --profile strict and --git-diff
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

# Same, with the strict profile (needs full git history)
./macnet scan --git-diff origin/main --profile strict --threshold 0.75
```

`--git-diff` compares your branch against the merge-base with the given branch, so changes that landed on the base branch after you branched are not counted as yours. Uncommitted changes are not included. Diff signals look at the lines your change added, not at the whole file.

**Local Installation**

You can also download the latest pre-compiled binaries from the [Releases](https://github.com/jakeeviado/macnet/releases) page and install them directly to your machine (supported on Linux & Windows), or clone the project and build it from source:

```bash
git clone https://github.com/jakeeviado/macnet.git
cd macnet
go build
```

---

### Disclaimer
MACnet is an aid for code quality and review; it is not intended to determine authorship or replace human judgment. Results are probabilistic and may include false positives. This applies to both profiles: `strict` reports more evidence, not a verdict, and its output should not be used as the sole basis for an academic, hiring, or disciplinary decision.

This is a hobby and personal project

### License
Licensed under the MIT License - see [LICENSE](LICENSE) file for details.

Open Sourced - You are free to fork, modify, distribute, and train your own ONNX model on this.
