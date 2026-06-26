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
	expect: { '': {} }
});

const callProbe = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'probe',
	params: [ 'name' ],
	expect: { '': {} }
});

const callServiceAction = rpc.declare({
	object: 'luci.failsafe-dns-proxy',
	method: 'setServiceAction',
	params: [ 'action' ],
	expect: { '': {} }
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

function messageOf(value, fallback) {
	if (value == null)
		return fallback || _('Unknown error');
	if (typeof value === 'string')
		return value || fallback || _('Unknown error');
	if (value.message)
		return value.message;
	if (value.error)
		return value.error;
	if (value.stderr)
		return value.stderr;
	if (value.stdout)
		return value.stdout;
	return fallback || _('Unknown error');
}

function resultOK(result) {
	return result && (result.result === true || result.result === 1 || result.result === '1' || result.success === true || result.success === 1 || result.success === '1');
}

function setLoading(button, loading, loadingText) {
	if (!button)
		return;
	if (loading) {
		document.querySelectorAll('.fdp-actions button,.fdp-section-actions button').forEach(function(item) {
			item.disabled = true;
		});
		button.dataset.fdpText = button.textContent;
		button.classList.add('spinning');
		button.textContent = loadingText || _('Working…');
	} else {
		button.classList.remove('spinning');
		button.disabled = false;
		if (button.dataset.fdpText) {
			button.textContent = button.dataset.fdpText;
			delete button.dataset.fdpText;
		}
	}
}

function actionHandler(buttonId, loadingText, action) {
	return function() {
		const button = document.getElementById(buttonId);
		setLoading(button, true, loadingText);
		return Promise.resolve()
			.then(action)
			.catch(function(error) {
				ui.addNotification(null, E('p', {}, messageOf(error, _('Operation failed'))), 'error');
			})
			.then(refreshRuntime)
			.finally(function() {
				setLoading(button, false);
				return refreshRuntime();
			});
	};
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

	const busy = document.querySelector('.fdp-actions .spinning,.fdp-section-actions .spinning');
	if (!busy) {
		setButtonDisabled('fdp-start', service.running || !service.config_enabled);
		setButtonDisabled('fdp-restart', !service.running || !service.config_enabled);
		setButtonDisabled('fdp-stop', !service.running);
		setButtonDisabled('fdp-enable-autostart', !!service.enabled);
		setButtonDisabled('fdp-disable-autostart', !service.enabled);
		setButtonDisabled('fdp-enable-integration', !!integration.enabled);
		setButtonDisabled('fdp-disable-integration', !integration.enabled);
	}
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
	const ok = resultOK(result);
	ui.addNotification(null, E('p', {}, ok ? messageOf(result, successText) : messageOf(result, _('Operation failed'))), ok ? 'info' : 'error');
	return refreshRuntime();
}

function notifyVerified(result, successText, verify, verifyFailedText) {
	const ok = resultOK(result);
	if (!ok) {
		ui.addNotification(null, E('p', {}, messageOf(result, _('Operation failed'))), 'error');
		return Promise.resolve();
	}
	return Promise.resolve(verify ? verify() : true).then(function(verified) {
		if (verified)
			ui.addNotification(null, E('p', {}, messageOf(result, successText)), 'info');
		else
			ui.addNotification(null, E('p', {}, verifyFailedText || _('The command completed, but the expected state was not observed.')), 'error');
	});
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
				'.fdp-actions{display:flex;flex-wrap:wrap;gap:.75rem;align-items:center;margin:1rem 0 1.25rem;padding:0 1rem}',
				'.fdp-section-actions{display:flex;flex-wrap:wrap;gap:.75rem;margin-top:1rem;padding:0 1rem 1rem}',
				'.fdp-runtime-section{padding:1rem 1rem 1.25rem}',
				'.fdp-integration-section{padding:1rem}',
				'.fdp-actions .btn,.fdp-section-actions .btn{min-width:9rem;white-space:normal}',
				'.fdp-log{box-sizing:border-box;width:100%;min-height:4.5em;padding:.75rem;margin-top:.5rem;border:1px solid var(--border-color-medium, #ddd);border-radius:4px;background:var(--background-color-low, #fff)}',
				'.fdp-help{margin:.25rem 1rem .75rem;color:var(--text-color-medium, #555)}'
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
						click: ui.createHandlerFn(this, actionHandler('fdp-start', _('Starting…'), function() {
							return callServiceAction('start').then(function(r) {
								return notifyVerified(r, _('Service started'), function() {
									return callService().then(function(service) { return !!service.running; });
								}, _('Service did not report as running after start.'));
							});
						}))
					}, _('Start')),
					E('button', {
						id: 'fdp-restart',
						class: 'btn cbi-button-action',
						click: ui.createHandlerFn(this, actionHandler('fdp-restart', _('Restarting…'), function() {
							return callServiceAction('restart').then(function(r) {
								return notifyVerified(r, _('Service restarted'), function() {
									return callService().then(function(service) { return !!service.running; });
								}, _('Service did not report as running after restart.'));
							});
						}))
					}, _('Restart')),
					E('button', {
						id: 'fdp-stop',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, actionHandler('fdp-stop', _('Stopping…'), function() {
							return callServiceAction('stop').then(function(r) {
								return notifyVerified(r, _('Service stopped'), function() {
									return callService().then(function(service) { return !service.running; });
								}, _('Service still reports as running after stop.'));
							});
						}))
					}, _('Stop')),
					E('button', {
						id: 'fdp-enable-autostart',
						class: 'btn cbi-button-positive',
						click: ui.createHandlerFn(this, actionHandler('fdp-enable-autostart', _('Applying…'), function() {
							return callServiceAction('enable').then(function(r) {
								return notifyVerified(r, _('Autostart enabled'), function() {
									return callService().then(function(service) { return !!service.enabled; });
								}, _('Autostart did not report as enabled.'));
							});
						}))
					}, _('Enable autostart')),
					E('button', {
						id: 'fdp-disable-autostart',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, actionHandler('fdp-disable-autostart', _('Applying…'), function() {
							return callServiceAction('disable').then(function(r) {
								return notifyVerified(r, _('Autostart disabled'), function() {
									return callService().then(function(service) { return !service.enabled; });
								}, _('Autostart still reports as enabled.'));
							});
						}))
					}, _('Disable autostart')),
					E('button', {
						id: 'fdp-check-config',
						class: 'btn cbi-button',
						click: ui.createHandlerFn(this, actionHandler('fdp-check-config', _('Checking…'), function() {
							return callCheckConfig().then(function(r) { return notifyResult(r, _('Configuration is valid')); });
						}))
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
			E('div', { class: 'cbi-section fdp-integration-section' }, [
				E('p', {}, _('Integration is explicit and reversible. The current dnsmasq configuration is backed up and automatically restored if verification fails.')),
				E('p', {}, [
					E('strong', { id: 'fdp-dnsmasq-status' }, '-')
				]),
				E('div', { class: 'fdp-section-actions' }, [
					E('button', {
						id: 'fdp-dry-run-integration',
						class: 'btn cbi-button',
						click: ui.createHandlerFn(this, actionHandler('fdp-dry-run-integration', _('Checking…'), function() {
							return callDnsmasqAction('dry-run').then(function(r) {
								return notifyResult(r, _('Preflight checks passed.'));
							});
						}))
					}, _('Dry run')),
					E('button', {
						id: 'fdp-enable-integration',
						class: 'btn cbi-button-positive',
						click: ui.createHandlerFn(this, actionHandler('fdp-enable-integration', _('Enabling…'), function() {
							return callDnsmasqAction('enable').then(function(r) {
								return notifyVerified(r, _('dnsmasq integration enabled.'), function() {
									return callDnsmasqIntegration().then(function(integration) { return !!integration.enabled; });
								}, _('dnsmasq integration did not report as enabled.'));
							});
						}))
					}, _('Enable integration')),
					E('button', {
						id: 'fdp-disable-integration',
						class: 'btn cbi-button-negative',
						click: ui.createHandlerFn(this, actionHandler('fdp-disable-integration', _('Restoring…'), function() {
							return callDnsmasqAction('disable').then(function(r) {
								return notifyVerified(r, _('dnsmasq integration disabled.'), function() {
									return callDnsmasqIntegration().then(function(integration) { return !integration.enabled; });
								}, _('dnsmasq integration still reports as enabled.'));
							});
						}))
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
