
from fastapi import FastAPI
import time

app = FastAPI()


@app.get("/{path:path}")
def catch_all(path: str):
    return {"path": path, "served_at": time.time()}
