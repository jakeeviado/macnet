## Introduction to Sentinel Machine Learning

This directory `root-dir/model` contains the Python training pipeline used to generate the ONNX-formatted machine learning model (`model.onnx`) consumed by the Go backend heuristic analysis engine.

---

**Before getting started...**

### Data Preparation
> Ensure to add more or customize the current sample human and flagged codes under the root `examples` directory.

> Ensure that your training data CSV (`sentinel_training.csv`), whether generated locally or via CI/CD pipelines, is placed directly inside this `model/` directory (or update the path in train.py to reference that CSV) before running the training script.

> A default sample **dumb model** (`model.onnx`) and training CSV (`sentinel_training.csv`) are included in this directory out-of-the-box for reference and immediate testing and bootstrapping.

### Prerequisites

Ensure you have Python 3.8+ installed, then install the required dependencies:

```bash
# 1. Change into the model directory
cd model

# 2. Install Python dependencies
pip install -r requirements.txt

# 3. Run the training script
python train.py
```

### Training & Model Generation
Run the training script to train the Random Forest Classifier, evaluate accuracy, and export the production-ready ONNX model:

```bash
python scripts/train.py
```

This will automatically output a new model.onnx file at the project root, synchronized with the current feature input dimension.

```bash
# Running script output
> python train.py
Converting model to ONNX format...
Training Complete. Accuracy: 100.00%
Production 'model.onnx' generated successfully.
```

**Upon successful execution, model.onnx is generated directly within this model/ directory. No manual file moving is required, as the Go backend is configured to read the model from this path by default.**
