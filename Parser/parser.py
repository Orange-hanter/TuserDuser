import requests

from classBresttheatre import BrestTheatre, ParseSite


def process_entity(site_parser: ParseSite):
    html_text = requests.get(site_parser.url).text
    site_parser.configure(html_text)
    return site_parser.extract_data()


if __name__ == "__main__":
    site = BrestTheatre()
    for i in process_entity(site):
        print(i, "\n")
