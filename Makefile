


.PHONY: sync
sync:
	rsync -avz --exclude='.git' --exclude='.claude' --exclude='.vscode' --exclude='.idea' --exclude='CLAUDE.md' . ubuntu@192.168.25.88:/home/ubuntu/zigo/go-micro
