import sys,re
src=open(sys.argv[1],encoding='utf-8',errors='replace').read()
pat=sys.argv[2]; before=int(sys.argv[3]); after=int(sys.argv[4]); maxn=int(sys.argv[5]) if len(sys.argv)>5 else 3
for i,m in enumerate(re.finditer(pat,src)):
    if i>=maxn: break
    print('---',m.start()); print(src[max(0,m.start()-before):m.end()+after])
