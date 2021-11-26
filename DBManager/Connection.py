import psycopg2


class DBConnection:
    def __init__(self):
        self._dbname = 'TuserDuser'
        self._user = 'postgres'
        self._password = '1111'
        self._host = 'localhost'
        self._port = 5432
        self._db_connector = None

    def connect(self):
        try:
            db_connector = psycopg2.connect(database=self._dbname,
                                            user=self._user,
                                            password=self._password,
                                            host=self._host,
                                            port=self._port)
            print("DB connected")
        except Exception as e:
            print("FAIL: DB not connected")
            print(str(e))


if __name__ == "__main__":
    dd = DBConnection()
    dd.connect()
