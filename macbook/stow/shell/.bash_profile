export GPG_TTY="$( tty )"

# --- MISCELLANEOUS
alias weather="curl http://wttr.in"
alias code="cd ~/Sites"
alias ppath="echo $PATH | tr ':' '\n'"
alias zrestart="exec zsh --login"
alias air="~/go/bin/air"
alias api="~/Sites/oullin/api"
alias web="~/Sites/oullin/web"
alias infra="~/Sites/oullin/infra"
alias dk-clear="docker container prune -f && \
        docker image prune -f && \
        docker volume prune -f && \
        docker network prune -f && \
        docker system prune -a --volumes -f && \
        docker ps -aq | xargs --no-run-if-empty docker stop && \
        docker ps -aq | xargs --no-run-if-empty docker rm && \
        docker ps"

# --- GIT
alias gs="git status"
alias gaa="git add ."
alias gcc='git commit -S --amend --no-edit'
alias gc="git commit -S -a -m"
alias gcm="git checkout main && git pull"
alias gcd="git checkout develop && git pull"
alias gcq="git checkout qa && git pull"
alias gl="git log --graph --pretty=format:'%Cred%h%Creset -%C(yellow)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' --abbrev-commit"
alias nah="git reset --hard && git clean -df"
alias wip="git add . && git commit -m 'format'"
alias gclean="git fetch -p"
alias gempty="git commit --allow-empty -m 'Empty - Commit'"
alias gfresh="git config --global pull.rebase true" #reconcile git diverged

# --- LARAVEL
alias a="php artisan"
alias tinker='php artisan tinker'
alias aperm="sudo chgrp -R www-data storage/ bootstrap/cache && sudo chmod -R ug+rwx storage bootstrap/cache"
alias alogs="rm -rf storage/logs/*.*"
alias acache="php artisan cache:clear && php artisan config:clear && php artisan clear-compiled && php artisan route:clear && php artisan view:clear"
alias asail='[ -f sail ] && sh sail || sh vendor/bin/sail'
alias amf="php artisan migrate:fresh"

# --- PHP
alias cda="composer dumpautoload -o"
alias phpini="php -i | grep php.ini"
alias uf="./vendor/bin/phpunit --filter="
alias u="./vendor/bin/phpunit"





