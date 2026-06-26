'use strict';
'require view';
'require form';
'require rpc';
'require poll';
'require ui';

const callStatus = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'getStatus',
	expect: { '': {} }
});

const callService = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'getService',
	expect: { '': {} }
});

const callLogs = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'getLogs',
	params: [ 'limit' ],
	expect: { lines: [] }
});

const callCheckConfig = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'checkConfig',
	expect: { result: false, message: '' }
});

const callProbe = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'probe',
	params: [ 'name' ],
	expect: { success: false }
});

const callServiceAction = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'setServiceAction',
	params: [ 'action' ],
	expect: { result: false, message: '' }
});

const callDnsmasqIntegration = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'getDnsmasqIntegration',
	expect: { '': {} }
});

const callDnsmasqAction = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'setDnsmasqIntegration',
	params: [ 'action' ],
	expect: { '': {} }
});

function stateLabel(state) {
	const labels = {
		unknown: _('Unknown'),
		healthy: _('Healthy'),
		suspect: _('Suspect'),
		down: _('Down'),
		recovering: _('Recovering')
	};
	return labels[state] || state || '-';
}

function formatTime(value) {
	if (!value || value.indexOf('0001-01-01') === 0)
		return '-';
	const date = new Date(value);
	return isNaN(date.getTime()) ? value : date.toLocaleString();
}

function setButtonDisabled(id, disabled) {
	const button = document.getElementById(id);
	if (button)
		button.disabled = !!disabled;
}

function firstStringArgument(args) {
	for (let i = 0; i < args.length; i++)
		if (typeof args[i] === 'string' && args[i])
			return args[i];
	return null;
}

function updateRuntime(status, service, logs, integration) {
	const serviceNode = document.getElementById('fdp-service-status');
	const activeNode = document.getElementById('fdp-active-upstream');
	const table = document.getElementById('fdp-upstream-status');
	const logNode = document.getElementById('fdp-log');
	const integrationNode = document.getElementById('fdp-dnsmasq-status');

	if (serviceNode)
		serviceNode.textContent = service.running
			? _('Running, version %s; autostart %s').format(service.version || '-', service.enabled ? _('enabled') : _('disabled'))
			: (service.config_enabled
				? _('Stopped; autostart %s').format(service.enabled ? _('enabled') : _('disabled'))
				: _('Stopped; startup disabled in configuration'));
	if (activeNode)
		activeNode.textContent = status.active_upstream || '-';
	if (table) {
		const rows = (status.upstreams || []).map(function(upstream) {
			return [
				upstream.name,
				upstream.endpoint,
				upstream.priority,
				stateLabel(upstream.state),
				upstream.consecutive_failures || 0,
				upstream.last_latency_ns ? (upstream.last_latency_ns / 1000000).toFixed(1) + ' ms' : '-',
				formatTime(upstream.last_success),
				upstream.last_error || '-'
			];
		});
		cbi_update_table(table, rows, E('em', {}, _('No runtime status available.')));
	}
	if (logNode)
		logNode.textContent = (logs.lines || []).join('\n') || _('No operational log entries.');
	if (integrationNode)
		integrationNode.textContent = integration.enabled
			? _('Enabled: dnsmasq uses %s').format(integration.server || '-')
			: _('Disabled: dnsmasq configuration is unchanged');

	setButtonDisabled('fdp-start', service.running || !service.config_enabled);
	setButtonDisabled('fdp-restart', !service.running || !service.config_enabled);
	setButtonDisabled('fdp-stop', !service.running);
	setButtonDisabled('fdp-enable-autostart', !!service.enabled);
	setButtonDisabled('fdp-disable-autostart', !service.enabled);
	setButtonDisabled('fdp-enable-integration', !!integration.enabled);
	setButtonDisabled('fdp-disable-integration', !integration.enabled);
}

function refreshRuntime() {
	return Promise.all([
		L.resolveDefault(callStatus(), {}),
		L.resolveDefault(callService(), {}),
		L.resolveDefault(callLogs(100), { lines: [] }),
		L.resolveDefault(callDnsmasqIntegration(), {})
	]).then(function(data) {
		updateRuntime(data[0], data[1], data[2], data[3]);
	});
}

function notifyResult(result, successText) {
	const ok = !!result.result;
	ui.addNotification(null, E('p', {}, ok ? (result.message || successText) : (result.message || _('Operation failed'))), ok ? 'info' : 'error');
	return refreshRuntime();
}

function normalizedProbeValues(value) {
	const values = Array.isArray(value) ? value : [ value ];
	return values.map(function(item) {
		return (item || '').trim();
	}).filter(function(item) {
		return item !== '';
	});
}

return view.extend({
	load: function() {
		return Promise.all([
			L.resolveDefault(callStatus(), {}),
			L.resolveDefault(callService(), {}),
			L.resolveDefault(callLogs(100), { lines: [] }),
			L.resolveDefault(callDnsmasqIntegration(), {})
		]);
	},

	render: function(data) {
		let m, s, o;

		m = new form.Map('failsafe-dns-proxy', _('Failsafe DNS Proxy'),
			_('Strict-priority DNS failover with active health checks and automatic failback.'));

		s = m.section(form.NamedSection, 'main', 'main', _('General settings'));
		s.addremove = false;

		o = s.option(form.Flag, 'enabled', _('Allow daemon to run'),
			_('When disabled, the init script exits without starting the daemon. Autostart is controlled in the runtime status panel.'));
		o.default = '0';
		o.rmempty = false;

		o = s.option(form.Value, 'listen_addr', _('Listen address'));
		o.datatype = "ipaddr('nomask')";
		o.default = '127.0.0.1';
		o.rmempty = false;

		o = s.option(form.Value, 'listen_port', _('Listen port'));
		o.datatype = 'port';
		o.default = '5359';
		o.rmempty = false;

		o = s.option(form.Value, 'attempt_timeout_ms', _('Attempt timeout (ms)'));
		o.datatype = 'range(50,10000)';
		o.default = '700';
		o.rmempty = false;

		o = s.option(form.Value, 'request_timeout_ms', _('Total request timeout (ms)'));
		o.datatype = 'range(50,30000)';
		o.default = '2000';
		o.rmempty = false;

		o = s.option(form.Value, 'health_interval_s', _('Health check interval (s)'));
		o.datatype = 'range(1,3600)';
		o.default = '5';
		o.rmempty = false;

		o = s.option(form.Value, 'fail_threshold', _('Failure threshold'));
		o.datatype = 'range(1,20)';
		o.default = '2';
		o.rmempty = false;

		o = s.option(form.Value, 'recover_threshold', _('Recovery threshold'));
		o.datatype = 'range(1,20)';
		o.default = '2';
		o.rmempty = false;

		o = s.option(form.Value, 'max_concurrent', _('Maximum concurrent requests'));
		o.datatype = 'range(1,4096)';
		o.default = '128';
		o.rmempty = false;

		o = s.option(form.DynamicList, 'probe', _('Probe questions'),
			_('Use the format domain:A or domain:AAAA.'));
		o.rmempty = false;
		o.validate = function(sectionId, value) {
			const currentValues = normalizedProbeValues(value);
			const values = typeof this.formvalue === 'function'
				? normalizedProbeValues(this.formvalue(sectionId))
				: normalizedProbeValues(value);
			if (currentValues.length === 0)
				return values.length > 0 ? true : _('Expected domain:A or domain:AAAA');
			for (let i = 0; i < currentValues.length; i++)
				if (!/^[^:\s]+:(A|AAAA)$/i.test(currentValues[i]))
					return _('Expected domain:A or domain:AAAA');
			return true;
		};

		s = m.section(form.GridSection, 'upstream', _('Upstreams'),
			_('Upstreams are tried by ascending numeric priority.'));
		s.addremove = true;
		s.anonymous = false;
		s.sortable = true;
		s.nodescriptions = true;
		s.sectiontitle = function(sectionId) {
			return sectionId;
		};

		o = s.option(form.Flag, 'enabled', _('Enabled'));
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'priority', _('Priority'));
		o.datatype = 'uinteger';
		o.default = '10';
		o.rmempty = false;

		o = s.option(form.ListValue, 'protocol', _('Protocol'));
		o.value('udp', 'UDP');
		o.value('tcp', 'TCP');
		o.default = 'udp';
		o.rmempty = false;

		o = s.option(form.Value, 'address', _('IP address'));
		o.datatype = "ipaddr('nomask')";
		o.rmempty = false;

		o = s.option(form.Value, 'port', _('Port'));
		o.datatype = 'port';
		o.default = '53';
		o.rmempty = false;

		o = s.option(form.Button, '_probe', _('Test'));
		o.inputstyle = 'apply';
		o.onclick = function() {
			const sectionId = firstStringArgument(arguments);
			if (!sectionId) {
				ui.addNotification(null, E('p', {}, _('Unable to determine upstream to test.')), 'error');
				return Promise.resolve();
			}
			return callProbe(sectionId).then(function(result) {
				const text = result.success
					? _('%s responded with %s in %d ms').format(result.upstream, result.rcode, result.latency_ms)
					: _('%s probe failed: %s').format(result.upstream || sectionId, result.error || _('Unknown error'));
				ui.addNotification(null, E('p', {}, text), result.success ? 'info' : 'error');
			}).catch(function(error) {
				ui.addNotification(null, E('p', {}, _('%s probe failed: %s').format(sectionId, error.message || error)), 'error');
			});
		};

		const runtime = E('div', { class: 'cbi-map' }, [
			E('style', {}, [
				'.fdp-actions{display:flex;flex-wrap:wrap;gap:.5rem;align-items:center;margin:.75rem 0 1rem}',
				'.fdp-section-actions{display:flex;flex-wrap:wrap;gap:.5rem;margin-top:.75rem}',
				'.fdp-runtime-section{padding-bottom:1rem}',
				'.fdp-log{box-sizing:border-box;width:100%;min-height:4.5em;padding:.75rem;margin-top:.5rem;border:1px solid var(--border-color-medium, #ddd);border-radius:4px;background:var(--background-color-low, #fff)}',
				'.fdp-help{margin:.25rem 0 .75rem;color:var(--text-color-medium, #555)}'
			].join('\n')),
			E('h2', {}, _('Runtime status')),
			E('div', { class: 'cbi-section fdp-runtime-section' }, [
				E('div', { class: 'cbi-value' }, [
					E('label', { class: 'cbi-value-title' }, _('Service')),
					E('div', { class: 'cbi-value-field' }, [
						E('strong', { id: 'fdp-service-status' }, '-')
					])
				]),
				E('div', { class: 'cbi-value' }, [
					E('label', { class: 'cbi-value-title' }, _('Active upstream')),
					E('div', { class: 'cbi-value-field' }, [
						E('strong', { id: 'fdp-active-upstream' }, '-')
					])
				]),
				E('p', { class: 'fdp-help' }, _('Start and stop control the current daemon process. Autostart controls whether OpenWrt starts it during boot.')),
				E('div', { class: 'fdp-actions' }, [
					E('button', {
						id: 'fdp-start',
						class: 'btn cbi-button-action',
						click: ui.createHandlerFn(this, function() { return callServiceAction('start').then(function(r) { return notifyResult(r, _('Service started')); }); })
					}, _('Start')),
					E('button', {
						id: 'fdp-restart',
						class: 'btn cbi-button-action',
						click: ui.createHandlerFn(this, function() { return callServiceAction('restart').then(function(r) { return notifyResult(r, _('Service restarted')); }); })
					}, _('Restart')),
					E('button', {
						id: 'fdp-stop',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, function() { return callServiceAction('stop').then(function(r) { return notifyResult(r, _('Service stopped')); }); })
					}, _('Stop')),
					E('button', {
						id: 'fdp-enable-autostart',
						class: 'btn cbi-button-positive',
						click: ui.createHandlerFn(this, function() { return callServiceAction('enable').then(function(r) { return notifyResult(r, _('Autostart enabled')); }); })
					}, _('Enable autostart')),
					E('button', {
						id: 'fdp-disable-autostart',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, function() { return callServiceAction('disable').then(function(r) { return notifyResult(r, _('Autostart disabled')); }); })
					}, _('Disable autostart')),
					E('button', {
						id: 'fdp-check-config',
						class: 'btn cbi-button',
						click: ui.createHandlerFn(this, function() { return callCheckConfig().then(function(r) { return notifyResult(r, _('Configuration is valid')); }); })
					}, _('Check configuration'))
				])
			]),
			E('div', { class: 'cbi-section' }, [
				E('table', { class: 'table', id: 'fdp-upstream-status' }, [
					E('tr', { class: 'tr table-titles' }, [
						E('th', { class: 'th' }, _('Name')),
						E('th', { class: 'th' }, _('Endpoint')),
						E('th', { class: 'th' }, _('Priority')),
						E('th', { class: 'th' }, _('State')),
						E('th', { class: 'th' }, _('Failures')),
						E('th', { class: 'th' }, _('Latency')),
						E('th', { class: 'th' }, _('Last success')),
						E('th', { class: 'th' }, _('Last error'))
					])
				])
			]),
			E('h3', {}, _('dnsmasq integration')),
			E('div', { class: 'cbi-section' }, [
				E('p', {}, _('Integration is explicit and reversible. The current dnsmasq configuration is backed up and automatically restored if verification fails.')),
				E('p', {}, [
					E('strong', { id: 'fdp-dnsmasq-status' }, '-')
				]),
				E('div', { class: 'fdp-section-actions' }, [
					E('button', {
						id: 'fdp-dry-run-integration',
						class: 'btn cbi-button',
						click: ui.createHandlerFn(this, function() {
							return callDnsmasqAction('dry-run').then(function(r) {
								ui.addNotification(null, E('p', {}, r.message || _('Preflight checks passed.')), r.result === false ? 'error' : 'info');
								return refreshRuntime();
							});
						})
					}, _('Dry run')),
					E('button', {
						id: 'fdp-enable-integration',
						class: 'btn cbi-button-positive',
						click: ui.createHandlerFn(this, function() {
							return callDnsmasqAction('enable').then(function(r) {
								ui.addNotification(null, E('p', {}, r.message || _('dnsmasq integration enabled.')), r.result === false ? 'error' : 'info');
								return refreshRuntime();
							});
						})
					}, _('Enable integration')),
					E('button', {
						id: 'fdp-disable-integration',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, function() {
							return callDnsmasqAction('disable').then(function(r) {
								ui.addNotification(null, E('p', {}, r.message || _('dnsmasq integration disabled.')), r.result === false ? 'error' : 'info');
								return refreshRuntime();
							});
						})
					}, _('Disable and restore'))
				])
			]),
			E('h3', {}, _('Operational log')),
			E('pre', {
				id: 'fdp-log',
				class: 'fdp-log',
				style: 'max-height:24em;overflow:auto;white-space:pre-wrap'
			}, _('Loading…'))
		]);

		poll.add(refreshRuntime, 5);

		return m.render().then(function(formNode) {
			const page = E([], [ runtime, formNode ]);
			window.setTimeout(function() {
				updateRuntime(data[0], data[1], data[2], data[3]);
			}, 0);
			return page;
		});
	}
});
