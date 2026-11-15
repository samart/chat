import sys,re
s=open('/private/tmp/claude-501/-Users-user-Development-other-project/459f0484-dabf-4eb9-807f-8143c0170df0/scratchpad/sdk-pkgs/v0.3.325/package/sdk.mjs').read()
a=int(sys.argv[1]);b=int(sys.argv[2])
t=s[a:b]
t=re.sub(r'(?=(async )?function [\w$]+\()', '\n', t)
t=t.replace('async \nfunction','\nasync function')
open(sys.argv[3],'w').write(t)
