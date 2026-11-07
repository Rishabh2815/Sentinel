"""
Sentinel ML scoring service.

Exposes POST /score, which takes lightweight per-request features and
returns a bot_score in [0, 1]. Kept dependency-light (scikit-learn +
FastAPI) so it starts fast and stays under the gateway's latency budget.
"""

import os
import pickle
from typing import Optional

import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel

MODEL_PATH = os.environ.get("MODEL_PATH", "model.pkl")

app = FastAPI(title="Sentinel ML Scorer")


class ScoreRequest(BaseModel):
    client_ip: str
    user_agent: str
    interval_ms: float = 0.0
    header_count: int = 0


class ScoreResponse(BaseModel):
    bot_score: float


def _load_model():
    if os.path.exists(MODEL_PATH):
        with open(MODEL_PATH, "rb") as f:
            return pickle.load(f)
    return None


model = _load_model()


def extract_features(req: ScoreRequest) -> np.ndarray:
    """
    Turn a request into the same feature vector used in train.py:
      [interval_ms, header_count, ua_length, ua_has_bot_keyword]
    """
    ua = req.user_agent or ""
    ua_len = len(ua)
    ua_bot_kw = 1.0 if any(k in ua.lower() for k in ["bot", "spider", "curl", "python-requests", "scrapy"]) else 0.0
    return np.array([[req.interval_ms, req.header_count, ua_len, ua_bot_kw]])


@app.post("/score", response_model=ScoreResponse)
def score(req: ScoreRequest):
    features = extract_features(req)

    if model is not None:
        # predict_proba returns [P(human), P(bot)]
        proba = model.predict_proba(features)[0][1]
        return ScoreResponse(bot_score=float(proba))

    # Fallback heuristic if no trained model is present yet (Day 1-2 stopgap)
    ua_bot_kw = features[0][3]
    header_count = features[0][1]
    heuristic = 0.0
    if ua_bot_kw:
        heuristic += 0.6
    if header_count < 4:
        heuristic += 0.3
    return ScoreResponse(bot_score=min(heuristic, 1.0))


@app.get("/health")
def health():
    return {"status": "ok", "model_loaded": model is not None}
