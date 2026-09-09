import json,pathlib
b=pathlib.Path('/tmp/nn-tracer-replay')
p=b/'project-a/producer.jsonl'
with p.open('a') as f:f.write(json.dumps({'type':'message','id':'later-root','timestamp':'2026-09-02T12:00:00Z','message':{'role':'assistant','content':'Later producer note: consumer revalidation is still not recorded here.'}})+'\n')
(b/'project-a/new-conversation.jsonl').write_text(json.dumps({'type':'session','version':3,'id':'fixture-new'})+'\n'+json.dumps({'type':'message','id':'new-q','message':{'role':'user','content':'A newly recorded independent packaging task.'}})+'\n'+json.dumps({'type':'message','id':'new-report','message':{'role':'assistant','content':'Packaging plan drafted. No execution or compatibility results recorded.'}})+'\n')
print('ADVANCED: appended producer; added project-a/new-conversation.jsonl')
