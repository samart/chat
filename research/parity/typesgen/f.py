import os,sys,re
s=open(os.path.join(os.path.dirname(os.path.abspath(__file__)), '../../../reference/sdk-pkgs/v0.3.325/package/sdk.mjs')).read()
mode=sys.argv[1]
if mode=='def':
  for n in sys.argv[2:]:
    for pat in ['function %s('%n,'var %s='%n,',%s='%n,' %s='%n]:
      i=s.find(pat)
      while i!=-1:
        print(n,pat,i); i=s.find(pat,i+1)
elif mode=='sl':
  a=int(sys.argv[2]);b=int(sys.argv[3]);print(s[a:a+b])
elif mode=='find':
  for m in re.finditer(sys.argv[2],s): print(m.start(), s[max(0,m.start()-80):m.start()+120].replace('\n',' '))
elif mode=='fn':
  n=int(sys.argv[2])
  for name in sys.argv[3:]:
    i=-1
    for pat in ['function %s('%name,'var %s='%name,',%s='%name,'let %s='%name]:
      i=s.find(pat)
      if i!=-1: break
    print('=====',name,i); print(s[i:i+n] if i!=-1 else 'NOTFOUND')
