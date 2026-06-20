# скрипты для just

alias rund := run-docker
alias downd := down-docker
alias reld := reload-docker
alias logd := logs-docker

default_compose_file := 'docker-compose.yml'
default_service := 'app'
venv := 'source ./.venv/bin/activate &&'
mig := 'main>demo'
default_py := '3.14'
default_profile_file := 'main.py'
default_msg := 'set tag to {{tag}}'

run:
    go run ./main.go

run-docker file=default_compose_file:
    just test
    docker-compose -f {{file}} up --build -d

down-docker:
    docker-compose down

reload-docker file=default_compose_file:
    docker-compose down
    just rund {{file}}

logs-docker:
    docker-compose logs --follow

test:
    cd /home/d506n/work/magutils-go && go test ./... -count=1

coverage:
    uv run pytest --cov --cov-report=html
    xdg-open "htmlcov/index.html"

push:
    just test
    git push

tag-push tag msg=default_msg:
    # just lint
    just test
    git add .
    git commit -m '{{msg}}'
    # uv run tag_upd.py -t v{{tag}} -m '{{msg}}'
    git push
    git tag v{{tag}}
    git push origin v{{tag}}

rundl file=default_compose_file:
    just rund {{file}}
    just logd

connect service=default_service:
    docker-compose exec {{service}} bash