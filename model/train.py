"""
MACnet Machine Learning Training Script
---------------------------------------------------------------------------
This script processes the heuristic features collected by the Go backend (produced CSV file)
and trains a Random Forest Classifier to distinguish between human-authored
and AI-generated code
"""

import pandas as pd
import numpy as np
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType

df = pd.read_csv('macnet_training.csv')

X = df.drop('label', axis=1).values.astype(np.float32)
y = df['label'].values

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

model = RandomForestClassifier(
    n_estimators=150,
    max_depth=12,
    min_samples_leaf=2,
    random_state=42
)
model.fit(X_train, y_train)

initial_type = [('float_input', FloatTensorType([None, 21]))]

options = {id(model): {'zipmap': False}}

print("Converting model to ONNX format...")
onx = convert_sklearn(
    model,
    initial_types=initial_type,
    options=options,
    target_opset=12
)

with open("model.onnx", "wb") as f:
    f.write(onx.SerializeToString())

print(f"Training Complete. Accuracy: {model.score(X_test, y_test)*100:.2f}%")
print("Production 'model.onnx' generated successfully.")
