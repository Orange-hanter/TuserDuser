import sqlite3


def init_db():
    conn = sqlite3.connect("db.db")
    cursor = conn.cursor()
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'events'(id TEXT UNIQUE, description TEXT, date DATETIME)""")

    conn.commit()
