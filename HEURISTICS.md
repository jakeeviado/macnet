## MACnet Heuristic Signals

## How to read this

- **ID:** `F` = file-level, `G` = git-diff.
- **Needs:** what the signal reads.
  - **File:** one file's content.
  - **Diff:** only the lines the change added.
  - **Repo:** the existing codebase as a baseline (style, manifests, naming).
  - **History:** git history. Not available on shallow clones.
  - **Author:** the same author's earlier work. Assessment use only.
- **Class:** what it measures.
  - **Lexical:** names and vocabulary.
  - **Stylistic:** formatting and comments.
  - **Structural:** code shape.
  - **Provenance:** evidence of tooling or mixed authorship.
  - **Dependency:** imports and libraries.
  - **Process:** how the change was made.
- **Profile:** `loose` runs signals that need File, Diff, or cheap Repo data. `strict` adds the rest.
- **Status:** **Active** is implemented. **Next** is planned soon. **Later** is a low-weight signal or ML feature. **Parked** is context only, or likely dropped.

Diff-level signals are **advisory**: they appear in the report as evidence for a reviewer and never fail a build.

---

## File-level signals

All are active in the `loose` profile and feed the ML model.

| ID  | Signal                       | Looks for                                 | Class      |
| :-: | :--------------------------- | :---------------------------------------- | :--------: |
| F1  | `comment_density`            | Comment-to-code ratio                     | Stylistic  |
| F2  | `generic_naming`             | Placeholder names (`temp`, `data`)        | Lexical    |
| F3  | `repetitive_patterns`        | Duplicated lines within a file            | Structural |
| F4  | `code_complexity`            | Control flow density                      | Structural |
| F5  | `formatting_consistency`     | Unusually uniform indentation             | Stylistic  |
| F6  | `comment_redundancy`         | Comments that mirror the code beside them | Stylistic  |
| F7  | `emoji_sentiment`            | Informal emojis                           | Stylistic  |
| F8  | `identifier_order`           | Perfectly alphabetical identifier blocks  | Lexical    |
| F9  | `defensive_ratio`            | Error handling vs functional logic        | Structural |

## Git-diff signals

| ID  | Signal                       | Looks for                                                           | Class      | Needs   | Profile | Status |
| :-: | :--------------------------- | :------------------------------------------------------------------ | :--------: | :-----: | :-----: | :----: |
| G1  | `tool_artifacts`             | `CLAUDE.md`, `AGENTS.md`, `.cursorrules`, etc. headers              | Provenance | Diff    | loose   | Next   |
| G2  | `hallucinated_imports`       | Imports not in the manifest, or manifest entries nothing imports    | Dependency | Diff    | loose   | Next   |
| G3  | `provenance_asymmetry`       | Large gap between per-file scores within one diff                   | Provenance | Diff    | loose   | Next   |
| G4  | `docstring_uniformity`       | Added functions all polished while the surrounding file is not      | Stylistic  | Diff    | loose   | Later  |
| G5  | `message_diff_mismatch`      | Commit message scale disagrees with diff size                       | Process    | Diff    | loose   | Parked |
| G6  | `scope_spread`               | One commit, many unrelated packages, same change shape              | Structural | Diff    | loose   | Parked |
| G7  | `test_mirroring`             | Tests mirroring the implementation one-to-one, no edge cases        | Structural | Diff    | loose   | Later  |
| G8  | `vocabulary_drift`           | New code names an existing concept differently than the repo        | Lexical    | Repo    | loose   | Next   |
| G9  | `style_drift`                | Indentation style that differs between files in the change          | Stylistic  | Diff    | strict  | Active |
| G10 | `naming_convention_mismatch` | `camelCase` in a `snake_case` repo, or the reverse                  | Lexical    | Repo    | loose   | Later  |
| G11 | `comment_style_delta`        | Comment density and punctuation vs the repo                         | Stylistic  | Repo    | strict  | Later  |
| G12 | `defensive_delta`            | Error-handling density of new code vs the repo                      | Structural | Repo    | strict  | Later  |
| G13 | `dependency_idiom`           | A new library for something the repo already solves                 | Dependency | Repo    | strict  | Later  |
| G14 | `addition_deletion_ratio`    | Large additions with almost no deletion                             | Process    | Diff    | strict  | Active |
| G15 | `rework_rate`                | Added lines never revised in later commits                          | Process    | History | strict  | Next   |
| G16 | `first_commit_dump`          | First commit holds most of the final code                           | Process    | History | strict  | Later  |
| G17 | `temporal_velocity`          | Bursts of commits seconds apart                                     | Process    | History | strict  | Parked |
| G18 | `author_committer_skew`      | Author vs committer date gaps and timezone changes                  | Process    | History | strict  | Parked |
| G19 | `message_style_drift`        | Commit message style vs the author's earlier commits                | Process    | Author  | strict  | Parked |
| G20 | `stylometry_delta`           | Line length, quote and blank-line habits vs the author's prior work | Stylistic  | Author  | strict  | Next   |
| G21 | `vocabulary_delta`           | Identifier overlap with the author's earlier code                   | Lexical    | Author  | strict  | Later  |

---

## Make it your own

MACnet is MIT-licensed, so you do not need these signals or this model. Fork it, drop the signals you disagree with, add your own, and train your own ONNX model on your own data. If you change the signals, set your own thresholds from your own false-positive measurements, not from the defaults.

To add a signal:

1. Write the check in `pkg/analyzer/signals/` with its own `SignalMeta`.
2. Register it in `FileChecks` or `DiffChecks` in `registry.go`.
3. File-level signals only: append its name to `signalRegistry` in `pkg/ml/features.go`, then retrain the model.

The registry tests fail if you miss a step.
