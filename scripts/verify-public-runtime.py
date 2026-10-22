"""Execute already-published binaries; never publish, build, or use account credentials."""
import hashlib, http.server, json, os, platform, shutil, subprocess, sys, tarfile, tempfile, threading, urllib.request, zipfile

sys.stdout.reconfigure(encoding="utf-8")
from pathlib import Path

VERSION = "1.8.2"
root = Path(tempfile.mkdtemp(prefix="getnote-public-runtime-"))
evidence = {"version": VERSION, "system": platform.system(), "machine": platform.machine(), "checks": [], "business_auth_tested": False}

def check(name, ok):
    evidence["checks"].append({"case": name, "passed": bool(ok)})
    Path("runtime-evidence.json").write_text(json.dumps(evidence, indent=2), encoding="utf-8")
    assert ok, name
    print("PASS " + name, flush=True)

goos, arch = os.environ["ACCEPTANCE_GOOS"], os.environ["ACCEPTANCE_GOARCH"]
expected_machine = {"amd64": ("x86_64", "amd64"), "arm64": ("aarch64", "arm64")}[arch]
check("native-runner-architecture", platform.machine().lower() in expected_machine)
check("native-runner-system", platform.system().lower() == {"darwin":"darwin", "linux":"linux", "windows":"windows"}[goos])
# A fixed invalid test credential ensures no developer or runner account can be used.
env = {k:v for k,v in os.environ.items() if not k.startswith("GETNOTE_")}
env.update(GETNOTE_API_KEY="runtime-acceptance-invalid-credential", GETNOTE_CLIENT_ID="getnote-cli")
requests = []
class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *_): pass
    def do_GET(self):
        requests.append(self.path)
        data = {"has_more":False, "hide_time":False, "tasks":[]}
        if "/note/detail" in self.path:
            data = {"note":{"note_id":"123", "title":"runtime fixture", "chapter_timeline":{"source":"summary_markdown_rules", "items":[]}}}
        self.send_response(200); self.send_header("Content-Type", "application/json"); self.end_headers()
        self.wfile.write(json.dumps({"success":True,"data":data,"request_id":"runtime-local-fixture"}).encode())
server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
threading.Thread(target=server.serve_forever, daemon=True).start()
env["GETNOTE_API_URL"] = "http://127.0.0.1:" + str(server.server_port)

def run(command, args, expected=0, live=False):
    current = dict(env)
    if live: current["GETNOTE_API_URL"] = "https://openapi.biji.com"
    proc = subprocess.run(command + args + ["-o", "json"], env=current, capture_output=True, text=True, encoding="utf-8", timeout=60)
    check(command[0].split(os.sep)[-1] + " " + " ".join(args[:2]) + " exit", proc.returncode == expected)
    return json.loads(proc.stdout or proc.stderr)

def verify(label, command):
    proc = subprocess.run(command+["version"], env=env, capture_output=True, text=True, timeout=30)
    check(label+" version", proc.returncode==0 and proc.stdout.strip().split()[-1].lstrip("v")==VERSION)
    b=run(command,["capabilities"])
    check(label+" chapters contract", "note chapters" in b["commands"]["notes"])
    check(label+" string ID contract", "data.id_str" in b["command_results"]["file-add"]["success_fields"])
    for name,args in [("spaces",["save","   "]),("unicode spaces",["save","\u3000\u00a0\t"]),("invalid month",["sprouts","2026-99"])]:
        count=len(requests); b=run(command,args,1)
        check(label+" "+name+" rejected before HTTP", b.get("success") is False and b["error"]["reason"]=="cli_error" and len(requests)==count)
    b=run(command,["sprouts","2026-10"])
    check(label+" valid month local HTTP", b.get("success") is True and requests[-1].endswith("month=2026-10"))
    b=run(command,["note","chapters","123"])
    check(label+" empty chapters local HTTP", b["data"]["chapter_timeline"]["items"]==[])
    b=run(command,["notes","--page-size","1"],1,True)
    check(label+" real production TLS and credential rejection", b.get("success") is False and bool(b.get("request_id")) and b["error"]["reason"]!="cli_error")

try:
    npm=shutil.which("npm")
    check("npm available", bool(npm))
    proc=subprocess.run([npm,"install","--prefix",str(root/"npm"),"@getnote/cli@"+VERSION,"--no-audit","--no-fund"], capture_output=True, text=True, timeout=180)
    check("public npm install", proc.returncode==0)
    wrapper=root/"npm/node_modules/@getnote/cli/bin/getnote.js"
    verify("npm", [shutil.which("node"),str(wrapper)])
    suffix="zip" if goos=="windows" else "tar.gz"
    name=f"getnote-cli_{VERSION}_{goos}_{arch}.{suffix}"
    base=f"https://github.com/iswalle/getnote-cli/releases/download/v{VERSION}/"
    archive=root/name
    cached=os.environ.get("ACCEPTANCE_ASSET_DIR")
    if cached:
        shutil.copyfile(Path(cached)/name,archive)
        checksums=(Path(cached)/"checksums.txt").read_text()
        evidence["github_asset_source"]="previously downloaded public asset"
    else:
        urllib.request.urlretrieve(base+name,archive)
        checksums=urllib.request.urlopen(base+"checksums.txt",timeout=45).read().decode()
        evidence["github_asset_source"]="fresh public download"
    expected=next(line.split()[0] for line in checksums.splitlines() if line.split()[-1]==name)
    check("public GitHub SHA256", hashlib.sha256(archive.read_bytes()).hexdigest()==expected)
    target=root/"github"; target.mkdir()
    if suffix=="zip":
        with zipfile.ZipFile(archive) as z:
            check("archive path", z.namelist()==["getnote.exe"])
            z.extractall(target)
    else:
        with tarfile.open(archive) as z:
            check("archive path", z.getnames()==["getnote"])
            z.extractall(target)
    binary=target/("getnote.exe" if goos=="windows" else "getnote")
    binary.chmod(0o755)
    verify("github",[str(binary)])
    evidence["passed"]=True
finally:
    server.shutdown()
    Path("runtime-evidence.json").write_text(json.dumps(evidence,indent=2),encoding="utf-8")
