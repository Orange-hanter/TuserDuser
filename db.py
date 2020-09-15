import sqlite3


def init_db():
    conn = sqlite3.connect("db.db")
    cursor = conn.cursor()
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'events'(id TEXT UNIQUE, description TEXT, date DATETIME)""")

    conn.commit()

    
def add_event(description, date, id):
    c.execute("INSERT INTO employeer (description,date,id) VALUES ('%s','%d','%i')" % (
    description, date, id))
    conn.commit()
