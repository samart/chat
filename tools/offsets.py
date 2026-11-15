import sys
b=open(sys.argv[1],'rb').read()
pats=[l.rstrip('\n') for l in open(sys.argv[2],encoding='utf-8') if l.strip()]
for p in pats:
    pb=p.encode()
    i=b.find(pb); c=b.count(pb)
    print(f"{i:>8}  x{c}  {p[:110]}")
