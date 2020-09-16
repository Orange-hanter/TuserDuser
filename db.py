import sqlite3
import datetime

conn = sqlite3.connect("./DB/db.db", check_same_thread=False)
print("DB connected")


def init_db():
    request = """CREATE TABLE IF NOT EXISTS events (
                                            id          INTEGER  UNIQUE
                                            PRIMARY KEY AUTOINCREMENT,
                                            description TEXT,
                                            date        DATETIME
                                            );"""
    cursor = conn.cursor()
    cursor.execute(request)
    conn.commit()
    #fillint the data storage by random test values
    #put_test_data_to_db()
    cursor.close()

def add_event_db(description, date):
    cursor = conn.cursor()
    cursor.execute(f"INSERT INTO events (description, date) VALUES ({description}, {date})")
    conn.commit()
    cursor.close()


def get_events_today_db():
    today = datetime.date.today()
    request = f"""SELECT * 
                    FROM events 
                    WHERE date = '{today}'
                """
    cursor = conn.cursor()
    cursor.execute(request)
    event_list = cursor.fetchall()
    cursor.close()
    return event_list


def put_test_data_to_db():
    date = datetime.date.today()
    with open("./DB/events.test", 'b') as file:
        for line in file.readline():
            description = line
            date += datetime.timedelta(days=1)
            cursor = conn.cursor()
            cursor.execute(f"INSERT INTO events (description, date) VALUES ('{description}', '{date}')")
        conn.commit()
    cursor.close()