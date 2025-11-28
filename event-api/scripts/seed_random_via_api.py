#!/usr/bin/env python3
"""
seed_random_via_api.py

Генерирует случайные события на год вперёд и отправляет их через POST
в `POST /v1/api/events` на указанный API URL.

Настройки через переменные окружения:
  API_URL - базовый URL (по умолчанию http://localhost:8080)
  ACCESS_TOKEN - если задан, используется напрямую
  ADMIN_EMAIL / ADMIN_PASSWORD - используются для login, если ACCESS_TOKEN не задан
  DAYS - сколько дней вперед генерировать (по умолчанию 365)
  PROB - вероятность создания события для каждого дня (0..1, по умолчанию 0.7)

Скрипт использует только стандартную библиотеку Python.
"""
import os
import sys
import json
import random
import urllib.request
import urllib.error
from datetime import datetime, timedelta, timezone


API_URL = os.environ.get("API_URL", "http://localhost:8080")
LOGIN_URL = API_URL.rstrip("/") + "/v1/api/auth/login"
EVENTS_URL = API_URL.rstrip("/") + "/v1/api/events"
REVIEW_URL_TPL = API_URL.rstrip("/") + "/v1/api/events/{id}/review"
ACCESS_TOKEN = os.environ.get("ACCESS_TOKEN", "")
ADMIN_EMAIL = os.environ.get("ADMIN_EMAIL", "admin@example.com")
ADMIN_PASSWORD = os.environ.get("ADMIN_PASSWORD", "")

DAYS = int(os.environ.get("DAYS", "365"))
PROB = float(os.environ.get("PROB", "0.7"))
AUTO_APPROVE = os.environ.get("AUTO_APPROVE", "true").lower() in ("1", "true", "yes")

TYPES = ["Конференция", "Встреча", "Воркшоп", "Хакатон", "Лекция", "Вечеринка", "Спорт"]
PLACES = ["Главный Офис", "Коворкинг \"Старт\"", "Zoom / Online", "Центральный Парк", "Арт-Кафе", "Стадион"]
PRICE = ["free", "paid", "donation"]
SPEAKERS = ["Иван Иванов", "Мария Петрова", "Алексей Смирнов", "Ольга Кузнецова", "Елена Сидорова"]
LANGS = ["ru", "en"]


def http_post(url, data, headers=None):
    body = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(url, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    if headers:
        for k, v in headers.items():
            req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.getcode(), json.load(resp)
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.load(e)
        except Exception:
            return e.code, {"error": str(e)}
    except Exception as e:
        return None, {"error": str(e)}


def login(email, password):
    payload = {"email": email, "password": password}
    code, resp = http_post(LOGIN_URL, payload)
    if code == 200 and isinstance(resp, dict) and resp.get("access_token"):
        return resp.get("access_token")
    # Sometimes the response contains nested structure (models.AuthResponse)
    if isinstance(resp, dict) and resp.get("user") and resp.get("access_token"):
        return resp.get("access_token")
    raise RuntimeError(f"login failed: {code} {resp}")


def gen_event_for_day(base_date):
    # choose start hour between 8 and 20
    start_hour = random.randint(8, 20)
    start_minute = random.choice([0, 15, 30, 45])
    duration = random.randint(30, 180)
    start = datetime(base_date.year, base_date.month, base_date.day, 0, 0, tzinfo=timezone.utc) + timedelta(hours=start_hour, minutes=start_minute)
    end = start + timedelta(minutes=duration)
    typ = random.choice(TYPES)
    place = random.choice(PLACES)
    price = random.choice(PRICE)
    need_reg = random.random() > 0.5
    tags = random.sample(["tech", "community", "networking", "training", "fundraising", "health"], k=2)
    price_amount = None
    if price == "paid":
        price_amount = random.choice([500, 1000, 1500, 2500])
    details = {
        "description": "Автоматически сгенерированное тестовое событие.",
        "capacity": random.randint(10, 200),
        "tags": ["test", "generated"] + tags,
        "speaker": random.choice(SPEAKERS),
        "language": random.choice(LANGS),
    }
    if price_amount:
        details["price"] = price_amount
    # occasionally add an external link
    if random.random() > 0.8:
        details["link"] = f"https://example.com/events/{random.randint(1000,9999)}"
    payload = {
        "start": start.isoformat(),
        "end": end.isoformat(),
        "type": typ,
        "place": place,
        "priceType": price,
        "duration": duration,
        "needReg": need_reg,
        "details": details,
    }
    return payload


def approve_event(event_id, headers):
    url = REVIEW_URL_TPL.format(id=event_id)
    payload = {"action": "approve"}
    code, resp = http_post(url, payload, headers=headers)
    return code, resp


def main():
    global ACCESS_TOKEN
    if not ACCESS_TOKEN:
        if not ADMIN_PASSWORD:
            print("ERROR: ADMIN_PASSWORD or ACCESS_TOKEN must be set in environment.")
            sys.exit(1)
        print(f"Logging in as {ADMIN_EMAIL}...", file=sys.stderr)
        ACCESS_TOKEN = login(ADMIN_EMAIL, ADMIN_PASSWORD)
        print("Got access token", file=sys.stderr)

    headers = {"Authorization": f"Bearer {ACCESS_TOKEN}"}

    now = datetime.now(timezone.utc)
    created = 0
    failed = 0
    for d in range(DAYS):
        day = now + timedelta(days=d)
        if random.random() > PROB:
            continue
        payload = gen_event_for_day(day)
        code, resp = http_post(EVENTS_URL, payload, headers=headers)
        if code in (200, 201):
            created += 1
            # try to extract id from response
            event_id = None
            if isinstance(resp, dict):
                event_id = resp.get("id") or resp.get("ID")
            if AUTO_APPROVE and event_id:
                a_code, a_resp = approve_event(event_id, headers)
                if a_code == 200:
                    if created % 20 == 0:
                        print(f"Created+approved {created} events...", file=sys.stderr)
                else:
                    print(f"Created event {event_id} but failed to approve: {a_code} {a_resp}", file=sys.stderr)
            else:
                if created % 20 == 0:
                    print(f"Created {created} events...", file=sys.stderr)
        else:
            failed += 1
            print(f"Failed to create event: {code} {resp}", file=sys.stderr)

    print(f"Finished. created={created}, failed={failed}")


if __name__ == "__main__":
    main()
