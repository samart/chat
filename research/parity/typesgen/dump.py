import os,sys,re
s=open(os.path.join(os.path.dirname(os.path.abspath(__file__)), '../../../reference/sdk-pkgs/v0.3.325/package/sdk.mjs')).read()
a=int(sys.argv[1]);b=int(sys.argv[2])
t=s[a:b]
t=re.sub(r'(?=(async )?function [\w$]+\()', '\n', t)
t=t.replace('async \nfunction','\nasync function')
open(sys.argv[3],'w').write(t)
