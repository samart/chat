import re,sys
src=open(sys.argv[1]).read()
lim=int(sys.argv[2]) if len(sys.argv)>2 else 160
def rep(m):
    body=m.group(0)[3:-2]
    lines=[re.sub(r'^\s*\*\s?','',l) for l in body.split('\n')]
    txt=' '.join(l.strip() for l in lines if l.strip()).strip()
    tags=re.findall(r'@(deprecated|alpha|beta|internal|hidden|experimental|default\S*)',txt)
    t=re.split(r'(?<=[.!?])\s',txt,1)[0]
    if len(t)>lim: t=t[:lim]+'…'
    if tags: t+=' ['+','.join('@'+x for x in sorted(set(tags)))+']'
    return '// '+t
out=re.sub(r'/\*\*.*?\*/',rep,src,flags=re.S)
out=re.sub(r'\n[ \t]*\n(?:[ \t]*\n)+','\n\n',out)
# number lines relative to the original? keep original line numbers by prefix
sys.stdout.write(out)
