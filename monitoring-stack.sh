#!/bin/bash

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
	echo -e "${BLUE}═══════════════════════════════════════════${NC}"
	echo -e "${BLUE}$1${NC}"
	echo -e "${BLUE}═══════════════════════════════════════════${NC}"
}

print_success() {
	echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
	echo -e "${RED}✗ $1${NC}"
}

print_info() {
	echo -e "${YELLOW}ℹ $1${NC}"
}

# Main script
case "${1:-help}" in
start)
	print_header "Starting Event API Stack with Monitoring"

	if [ -f "event-api/docker-compose.yml" ]; then
		docker-compose -f event-api/docker-compose.yml up -d
		print_success "Stack started"

		sleep 3

		print_info "Waiting for services to be ready..."
		sleep 5

		echo ""
		print_header "Service URLs"
		echo -e "${GREEN}Event API${NC}      http://localhost:8080"
		echo -e "${GREEN}Prometheus${NC}     http://localhost:9090"
		echo -e "${GREEN}Grafana${NC}        http://localhost:3000 (admin/admin)"
		echo -e "${GREEN}Loki${NC}           http://localhost:3100"
		echo ""

		docker-compose -f docker-compose.full.yml ps
	else
		print_error "docker-compose.full.yml not found"
		exit 1
	fi
	;;

stop)
	print_header "Stopping Event API Stack"
	docker-compose -f event-api/docker-compose.yml down
	print_success "Stack stopped"
	;;

restart)
	print_header "Restarting Event API Stack"
	docker-compose -f event-api/docker-compose.yml restart
	print_success "Stack restarted"
	;;

logs)
	service="${2:-event-api-app}"
	print_header "Logs for $service"
	docker-compose -f event-api/docker-compose.yml logs -f "$service"
	;;

status)
	print_header "Stack Status"
	docker-compose -f event-api/docker-compose.yml ps
	;;

clean)
	print_header "Cleaning Stack (removing volumes)"
	read -p "Are you sure? This will delete all data. (y/N) " -n 1 -r
	echo
	if [[ $REPLY =~ ^[Yy]$ ]]; then
		docker-compose -f event-api/docker-compose.yml down -v
		print_success "Stack cleaned"
	else
		print_info "Cancelled"
	fi
	;;

health)
	print_header "Health Check"

	print_info "Checking Event API..."
	if curl -s http://localhost:8080/health >/dev/null; then
		print_success "Event API is healthy"
	else
		print_error "Event API is not responding"
	fi

	print_info "Checking Prometheus..."
	if curl -s http://localhost:9090/-/healthy >/dev/null; then
		print_success "Prometheus is healthy"
	else
		print_error "Prometheus is not responding"
	fi

	print_info "Checking Grafana..."
	if curl -s http://localhost:3000/api/health >/dev/null; then
		print_success "Grafana is healthy"
	else
		print_error "Grafana is not responding"
	fi

	print_info "Checking Loki..."
	if curl -s http://localhost:3100/ready >/dev/null; then
		print_success "Loki is healthy"
	else
		print_error "Loki is not responding"
	fi
	;;

*)
	print_header "Event API Monitoring Stack"
	echo ""
	echo "Usage: $0 {start|stop|restart|logs|status|clean|health}"
	echo ""
	echo "Commands:"
	echo "  start       - Start the full stack"
	echo "  stop        - Stop the stack"
	echo "  restart     - Restart the stack"
	echo "  logs        - Show logs for a service (default: event-api-app)"
	echo "  status      - Show stack status"
	echo "  clean       - Stop and remove all volumes"
	echo "  health      - Check health of all services"
	echo ""
	echo "Examples:"
	echo "  $0 start"
	echo "  $0 logs prometheus"
	echo "  $0 logs event-api-app"
	echo ""
	;;
esac
