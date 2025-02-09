function llm-shell
    # --- 1. Configuration ---
    set -l history_file "$HOME/.config/fish/functions/llm_history"
    set -l api_endpoint "http://127.0.0.1:9003/proxy/8000/" #  Use the local server

    # --- 2. Gather System Information ---
    set -l system_info (fastfetch | string collect)
    set -l packages (paru -Qq | string join "\n")

    # --- 3. Main Loop ---
    while true
        # --- 4. Get User Input ---
        read -l -P "Your request (or 'exit' to quit): " user_request

        if string match -i exit "$user_request"
            break
        end

        # --- 5. Load History ---
        set -l history_rec ""
        if test -f "$history_file"
            set -l history_rec (cat "$history_file" | string collect)
        end

        # --- 6. Construct the Request Body (as JSON) ---
        set -l request_body (jq -n \
            --arg input "$user_request" \
            --arg system_info "$system_info" \
            --arg history_file "$history_rec" \
            --arg packages "$packages" \
            '{input: $input, system_info: $system_info, history_file: $history_file, packages: $packages}')


        # --- 7. Call the Go Server (using curl, POST, streaming) ---
        set -l api_response (curl --no-buffer -s -X POST \
            -H "Content-Type: application/json" \
            -d "$request_body" \
            "$api_endpoint")

        if test -z "$api_response"
            echo "Error: No response from API."
            continue
        end

        if string match -r '^Error:' "$api_response"
            echo "$api_response"
            continue
        end

        # --- 8. Execute command and handle output ---
        set -l trimmed_command (string trim "$api_response")

        if test -n "$trimmed_command"
            echo "LLM suggests: $trimmed_command"
            read -l -P "Execute this command? (y/N) " confirm
            if string match -i y "$confirm"
                set -l cmd_output (eval $trimmed_command 2>&1)
                echo "$cmd_output"
                echo "" >>"$history_file"
                echo --- >>"$history_file"
                echo "User Request: $user_request" >>"$history_file"
                echo "Command: $trimmed_command" >>"$history_file"
                echo "Output:" >>"$history_file"
                echo "$cmd_output" >>"$history_file"
                echo --- >>"$history_file"
            else
                echo "Command not executed."
            end
        else
            echo "No command suggested."
        end
    end

    echo "Exiting llm-shell."
end
