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
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'users'(id TEXT UNIQUE)""")
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'tasklist'
                                  (id text, number text, time text, text text, uid text)
                               """)
    conn.commit()
    # filling the data storage by random test values
    # put_test_data_to_db()
    cursor.close()


def add_event_db(description, date):
    cursor = conn.cursor()
    cursor.execute(f"INSERT INTO events (description, date) VALUES ('{description}', '{date}')")
    conn.commit()
    cursor.close()


def get_events_by_day_db(date):
    request = f"""SELECT * 
                        FROM events 
                        WHERE date = '{date}'
                    """
    cursor = conn.cursor()
    cursor.execute(request)
    event_list = cursor.fetchall()
    cursor.close()
    return event_list


def get_events_today_db():
    return get_events_by_day_db(datetime.date.today())


def get_events_by_period_db(date_start, date_end):
    return []


def put_test_data_to_db():
    date = datetime.date.today()
    with open("./DB/events.test", 'r') as file:
        for line in file.readlines():
            description = line
            date += datetime.timedelta(days=1)
            cursor = conn.cursor()
            cursor.execute(f"INSERT INTO events (description, date) VALUES ('{description}', '{date}')")
        conn.commit()
    cursor.close()

def add_to_db_tasklist(chatid, number, time, text, uid):  # Функция добавляет данные в таблицу 'tasklist'
    conn = sqlite3.connect("./DB/db.db")
    cursor = conn.cursor()
    ins = f"""INSERT INTO 'tasklist'  VALUES ('{chatid}', '{number}', '{time}', '{text}', '{uid}')"""
    cursor.execute(ins)
    conn.commit()

if __name__ == '__main__':
    # put_test_data_to_db()
    add_event_db("Надеюсь получить фидбек :3 ", datetime.date.today())