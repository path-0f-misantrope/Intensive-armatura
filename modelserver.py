import os
import joblib
import numpy as np
import pandas as pd
import matplotlib
matplotlib.use('Agg')  # Используем безголовый бэкенд для Matplotlib
import matplotlib.pyplot as plt

import base64
import io
from flask import Flask, request, jsonify

app = Flask(__name__)

import sys


if getattr(sys, 'frozen', False): 
    BASE_DIR = sys._MEIPASS
else:
    BASE_DIR = os.path.dirname(os.path.abspath(__file__))

model_path = os.path.join(BASE_DIR, "sarimax_model.joblib")



model = joblib.load(model_path)


@app.route("/predict", methods=["POST"])
def predict():
    if "file" not in request.files:
        return jsonify({"error": "CSV file is required"}), 400

    file = request.files["file"]
    df_test = pd.read_csv(file, parse_dates=["dt"])
    df_test.set_index("dt", inplace=True)
    df_test.drop(columns=['month'], inplace=True)
    # да да признаю оно захардкожено под один дата сет, но я не успеваю и устал соберать еще какие то для тестов. давай представим что туда сразу хорошие данные приходят)))))
    exog_test = df_test.drop(columns=["Цена на арматуру"], errors="ignore")
    predicted_values = model.predict(start=df_test.index[0], end=df_test.index[-1], exog=exog_test)
    df_test["predict"] = predicted_values


    plt.figure(figsize=(10, 5))

    plt.plot(df_test.index, df_test["predict"], label='Прогноз', color='blue', linestyle='dashed')
    plt.title('Прогноз цены на арматуру')
    plt.xlabel('Дата')
    plt.ylabel('Цена')
    plt.legend()


    img_buffer = io.BytesIO()
    plt.savefig(img_buffer, format="png")
    plt.close()
    img_buffer.seek(0)

 
    img_base64 = base64.b64encode(img_buffer.read()).decode("utf-8")

    return jsonify({
        "dates": df_test.index.strftime("%Y-%m-%d").tolist(),
        "prediction": df_test["predict"].round(2).tolist(),
        "image_base64": img_base64
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)