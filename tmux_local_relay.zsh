#!/bin/zsh

# Capture the current directory
current_dir=$(pwd)

# Start a new tmux session but don't attach to it yet
tmux new-session -d -s relay -c "$current_dir"

# Split the window into three panes, making sure to set the directory
tmux split-window -h -c "$current_dir"
tmux split-window -v -c "$current_dir"

# Optionally, you can balance the splits
tmux select-layout even-horizontal

# Pane indexes start at 0 by default for the first window, incremented thereafter

# Start the target server to log relayed requests in the third pane
tmux send-keys -t relay:0.2 "cd \"$current_dir\"" C-m
tmux send-keys -t relay:0.2 "clear" C-m
tmux send-keys -t relay:0.2 '/Users/clint/src/fsdev/tools/python/bin/python simple_http_server.py 8085' C-m

# Make and run the local relay server in the second pane
tmux send-keys -t relay:0.1 "cd \"$current_dir\"" C-m
tmux send-keys -t relay:0.1 "clear" C-m
tmux send-keys -t relay:0.1 'make' C-m
tmux send-keys -t relay:0.1 'TRAFFIC_RELAY_TARGET=http://localhost:8085 TRAFFIC_EXCLUDE_BODY_CONTENT=Args ./dist/relay' C-m

# Use cURL to send a POST request to the relay server in the first pane
tmux send-keys -t relay:0.0 "cd \"$current_dir\"" C-m
tmux send-keys -t relay:0.0 "clear" C-m
# print the cURL command to the pane but don't run it
tmux send-keys -t relay:0.0 'curl POST -d "{"When":1000,"Seq":1,"Evts":[{"When":1000,"Kind":2,"Args":[]}]}" http://localhost:8990/rec/bundle'
tmux select-pane -t relay:0.0

# Finally, attach to the session
tmux attach-session -t relay