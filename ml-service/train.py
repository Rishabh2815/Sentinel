"""
Trains a small gradient boosting classifier on synthetic request features
and writes model.pkl for app.py to load.

Replace generate_synthetic_data() with real logged traffic (from the
Redis Streams event log) once the gateway has been running for a while.

Usage:
    python train.py
"""

import pickle

import numpy as np
from sklearn.ensemble import GradientBoostingClassifier
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report


def generate_synthetic_data(n=5000, seed=42):
    rng = np.random.default_rng(seed)
    n_bots = n // 3
    n_humans = n - n_bots

    # Humans: irregular timing, more headers, normal UA length
    human_interval = rng.normal(800, 400, n_humans).clip(50, 5000)
    human_headers = rng.integers(6, 15, n_humans)
    human_ua_len = rng.normal(90, 15, n_humans).clip(30, 150)
    human_bot_kw = np.zeros(n_humans)

    # Bots: very regular timing, fewer headers, shorter/scripted UA, some flag keywords
    bot_interval = rng.normal(120, 30, n_bots).clip(10, 400)
    bot_headers = rng.integers(2, 6, n_bots)
    bot_ua_len = rng.normal(40, 10, n_bots).clip(10, 80)
    bot_bot_kw = rng.choice([0, 1], size=n_bots, p=[0.4, 0.6])

    X = np.vstack([
        np.column_stack([human_interval, human_headers, human_ua_len, human_bot_kw]),
        np.column_stack([bot_interval, bot_headers, bot_ua_len, bot_bot_kw]),
    ])
    y = np.concatenate([np.zeros(n_humans), np.ones(n_bots)])

    return X, y


def main():
    X, y = generate_synthetic_data()
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

    clf = GradientBoostingClassifier(n_estimators=100, max_depth=3, random_state=42)
    clf.fit(X_train, y_train)

    y_pred = clf.predict(X_test)
    print(classification_report(y_test, y_pred, target_names=["human", "bot"]))

    with open("model.pkl", "wb") as f:
        pickle.dump(clf, f)
    print("saved model.pkl")


if __name__ == "__main__":
    main()
