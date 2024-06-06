#!/usr/bin/env python
import html, http.client, http.server, io, json, os, pickle, random, re, socket, socketserver, sqlite3, string, sys, subprocess, time, traceback, urllib.parse, urllib.request, xml.etree.ElementTree  # Python 3 required
try:
    import lxml.etree
except ImportError:
    print("[!] please install 'python-lxml' to (also) get access to XML vulnerabilities (e.g. '%s')\n" % ("apt-get install python-lxml" if os.name != "nt" else "https://pypi.python.org/pypi/lxml"))

NAME, VERSION, GITHUB, AUTHOR, LICENSE = "Damn Small Vulnerable Web (DSVW) < 100 LoC (Lines of Code)", "0.2b", "https://github.com/stamparm/DSVW", "Miroslav Stampar (@stamparm)", "Unlicense (public domain)"
LISTEN_ADDRESS, LISTEN_PORT = "127.0.0.1", 65412
HTML_PREFIX, HTML_POSTFIX = "<!DOCTYPE html>\n<html>\n<head>\n<style>a {font-weight: bold; text-decoration: none; visited: blue; color: blue;} ul {display: inline-block;} .disabled {text-decoration: line-through; color: gray} .disabled a {visited: gray; color: gray; pointer-events: none; cursor: default} table {border-collapse: collapse; margin: 12px; border: 2px solid black} th, td {border: 1px solid black; padding: 3px} span {font-size: larger; font-weight: bold}</style>\n<title>%s</title>\n</head>\n<body style='font: 12px monospace'>\n<script>function process(data) {alert(\"Surname(s) from JSON results: \" + Object.keys(data).map(function(k) {return data[k]}));}; var index=document.location.hash.indexOf('lang='); if (index != -1) document.write('<div style=\"position: absolute; top: 5px; right: 5px;\">Chosen language: <b>' + decodeURIComponent(document.location.hash.substring(index + 5)) + '</b></div>');</script>\n" % html.escape(NAME), "<div style=\"position: fixed; bottom: 5px; text-align: center; width: 100%%;\">Powered by <a href=\"%s\" style=\"font-weight: bold; text-decoration: none; visited: blue; color: blue\" target=\"_blank\">%s</a> (v<b>%s</b>)</div>\n</body>\n</html>" % (GITHUB, re.search(r"\(([^)]+)", NAME).group(1), VERSION)
USERS_XML = """<?xml version="1.0" encoding="utf-8"?><users><user id="0"><username>admin</username><name>admin</name><surname>admin</surname><password>7en8aiDoh!</password></user><user id="1"><username>dricci</username><name>dian</name><surname>ricci</surname><password>12345</password></user><user id="2"><username>amason</username><name>anthony</name><surname>mason</surname><password>gandalf</password></user><user id="3"><username>svargas</username><name>sandra</name><surname>vargas</surname><password>phest1945</password></user></users>"""

def init():
    global connection
    http.server.HTTPServer.allow_reuse_address = True
    connection = sqlite3.connect(":memory:", isolation_level=None, check_same_thread=False)
    cursor = connection.cursor()
    cursor.execute("CREATE TABLE users(id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, name TEXT, surname TEXT, password TEXT)")
    cursor.executemany("INSERT INTO users(id, username, name, surname, password) VALUES(NULL, ?, ?, ?, ?)", ((_.findtext("username"), _.findtext("name"), _.findtext("surname"), _.findtext("password")) for _ in xml.etree.ElementTree.fromstring(USERS_XML).findall("user")))
    cursor.execute("CREATE TABLE comments(id INTEGER PRIMARY KEY AUTOINCREMENT, comment TEXT, time TEXT)")

class ReqHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        path, query = self.path.split('?', 1) if '?' in self.path else (self.path, "")
        code, content, params, cursor = http.client.OK, HTML_PREFIX, dict((match.group("parameter"), urllib.parse.unquote(','.join(re.findall(r"(?:\A|[?&])%s=([^&]+)" % match.group("parameter"), query)))) for match in re.finditer(r"((\A|[?&])(?P<parameter>[\w\[\]]+)=)([^&]+)", query)), connection.cursor()
        try:
            if path == '/':
                if "id" in params:
                    cursor.execute("SELECT id, username, name, surname FROM users WHERE id=" + params["id"])
                    content += "<div><span>Result(s):</span></div><table><thead><th>id</th><th>username</th><th>name</th><th>surname</th></thead>%s</table>%s" % ("".join("<tr>%s</tr>" % "".join("<td>%s</td>" % ("-" if _ is None else _) for _ in row) for row in cursor.fetchall()), HTML_POSTFIX)
                elif "v" in params:
                    content += re.sub(r"(v<b>)[^<]+(</b>)", r"\g<1>%s\g<2>" % params["v"], HTML_POSTFIX)
                elif "object" in params:
                    content = str(pickle.loads(params["object"].encode()))
                elif "path" in params:
                    content = (open(os.path.abspath(params["path"]), "rb") if not "://" in params["path"] else urllib.request.urlopen(params["path"])).read().decode()
                elif "domain" in params:
                    content = subprocess.check_output("nslookup " + params["domain"], shell=True, stderr=subprocess.STDOUT, stdin=subprocess.PIPE).decode()
                elif "xml" in params:
                    content = lxml.etree.tostring(lxml.etree.parse(io.BytesIO(params["xml"].encode()), lxml.etree.XMLParser(no_network=False)), pretty_print=True).decode()
                elif "name" in params:
                    found = lxml.etree.parse(io.BytesIO(USERS_XML.encode())).xpath(".//user[name/text()='%s']" % params["name"])
                    content += "<b>Surname:</b> %s%s" % (found[-1].find("surname").text if found else "-", HTML_POSTFIX)
                elif "size" in params:
                    start, _ = time.time(), "<br>".join("#" * int(params["size"]) for _ in range(int(params["size"])))
                    content += "<b>Time required</b> (to 'resize image' to %dx%d): %.6f seconds%s" % (int(params["size"]), int(params["size"]), time.time() - start, HTML_POSTFIX)
                elif "comment" in params or query == "comment=":
                    if "comment" in params:
                        cursor.execute("INSERT INTO comments VALUES(NULL, '%s', '%s')" % (params["comment"], time.ctime()))
                        content += "Thank you for leaving the comment. Please click here <a href=\"/?comment=\">here</a> to see all comments%s" % HTML_POSTFIX
                    else:
                        cursor.execute("SELECT id, comment, time FROM comments")
                        content += "<div><span>Comment(s):</span></div><table><thead><th>id</th><th>comment</th><th>time</th></thead>%s</table>%s" % ("".join("<tr>%s</tr>" % "".join("<td>%s</td>" % ("-" if _ is None else _) for _ in row) for row in cursor.fetchall()), HTML_POSTFIX)
                elif "include" in params:
                    backup, sys.stdout, program, envs = sys.stdout, io.StringIO(), (open(params["include"], "rb") if not "://" in params["include"] else urllib.request.urlopen(params["include"])).read(), {"DOCUMENT_ROOT": os.getcwd(), "HTTP_USER_AGENT": self.headers.get("User-Agent"), "REMOTE_ADDR": self.client_address[0], "REMOTE_PORT": self.client_address[1], "PATH": path, "QUERY_STRING": query}
                    exec(program, envs)
                    content += sys.stdout.getvalue()
                    sys.stdout = backup
                elif "redir" in params:
                    content = content.replace("<head>", "<head><meta http-equiv=\"refresh\" content=\"0; url=%s\"/>" % params["redir"])
                if HTML_PREFIX in content and HTML_POSTFIX not in content:
                    content += "<div><span>Attacks:</span></div>\n<ul>%s\n</ul>\n" % ("".join("\n<li%s>%s - <a href=\"%s\">vulnerable</a>|<a href=\"%s\">exploit</a>|<a href=\"%s\" target=\"_blank\">info</a></li>" % (" class=\"disabled\" title=\"module 'python-lxml' not installed\"" if ("lxml.etree" not in sys.modules and any(_ in case[0].upper() for _ in ("XML", "XPATH"))) else "", case[0], case[1], case[2], case[3]) for case in CASES)).replace("<a href=\"None\">vulnerable</a>|", "<b>-</b>|")
            elif path == "/users.json":
                content = "%s%s%s" % ("" if not "callback" in params else "%s(" % params["callback"], json.dumps(dict((_.findtext("username"), _.findtext("surname")) for _ in xml.etree.ElementTree.fromstring(USERS_XML).findall("user"))), "" if not "callback" in params else ")")
            elif path == "/login":
                cursor.execute("SELECT * FROM users WHERE username='" + re.sub(r"[^\w]", "", params.get("username", "")) + "' AND password='" + params.get("password", "") + "'")
                content += "Welcome <b>%s</b><meta http-equiv=\"Set-Cookie\" content=\"SESSIONID=%s; path=/\"><meta http-equiv=\"refresh\" content=\"1; url=/\"/>" % (re.sub(r"[^\w]", "", params.get("username", "")), "".join(random.sample(string.ascii_letters + string.digits, 20))) if cursor.fetchall() else "The username and/or password is incorrect<meta http-equiv=\"Set-Cookie\" content=\"SESSIONID=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT\">"
            else:
                code = http.client.NOT_FOUND
        except Exception as ex:
            content = ex.output if isinstance(ex, subprocess.CalledProcessError) else traceback.format_exc()
            code = http.client.INTERNAL_SERVER_ERROR
        finally:
            self.send_response(code)
            self.send_header("Connection", "close")
            self.send_header("X-XSS-Protection", "0")
            self.send_header("Content-Type", "%s%s" % ("text/html" if content.startswith("<!DOCTYPE html>") else "text/plain", "; charset=%s" % params.get("charset", "utf8")))
            self.end_headers()
            self.wfile.write(("%s%s" % (content, HTML_POSTFIX if HTML_PREFIX in content and GITHUB not in content else "")).encode())
            self.wfile.flush()

class ThreadingServer(socketserver.ThreadingMixIn, http.server.HTTPServer):
    def server_bind(self):
        self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        http.server.HTTPServer.server_bind(self)

if __name__ == "__main__":
    init()
    print("%s #v%s\n by: %s\n\n[i] running HTTP server at 'http://%s:%d'..." % (NAME, VERSION, AUTHOR, LISTEN_ADDRESS, LISTEN_PORT))
    try:
        ThreadingServer((LISTEN_ADDRESS, LISTEN_PORT), ReqHandler).serve_forever()
    except KeyboardInterrupt:
        pass
    except Exception as ex:
        print("[x] exception occurred ('%s')" % ex)
    finally:
        os._exit(0)
