import React from "react";
import { AttributionControl, LngLatBounds, Map as MapLibreMap, Marker, NavigationControl, Popup, setWorkerUrl } from "maplibre-gl";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";
import { Crosshair, Map as MapIcon, MapPin, Navigation, RadioTower, Save, X } from "lucide-react";
import "maplibre-gl/dist/maplibre-gl.css";
import { Device, FleetData, updateDevice } from "./api";

type Props = { data: FleetData; onData: (data: FleetData) => void; onRemote: (device: Device) => void };
const OPEN_STREET_MAP_STYLE = "https://tiles.openfreemap.org/styles/liberty";

setWorkerUrl(workerUrl);

export function FleetMap({ data, onData, onRemote }: Props) {
  const [filter, setFilter] = React.useState<"all" | "connected" | "offline">("all");
  const [editing, setEditing] = React.useState<Device | null>(null);
  const [selected, setSelected] = React.useState<Device | null>(null);
  const [selectionRevision, setSelectionRevision] = React.useState(0);
  const located = React.useMemo(() => data.devices.filter((device) => device.location && (filter === "all" || device.presence === filter)), [data.devices, filter]);
  const connected = data.devices.filter((device) => device.presence === "connected").length;
  const selectDevice = React.useCallback((device: Device) => {
    setSelected(device);
    setSelectionRevision((revision) => revision + 1);
  }, []);
  React.useEffect(() => {
    if (selected && !located.some((device) => device.id === selected.id)) setSelected(null);
  }, [located, selected]);
  return <>
    <section className="mapStats">
      <div><span className="mapStatIcon green"><RadioTower size={18} /></span><strong>{connected}</strong><small>Online</small></div>
      <div><span className="mapStatIcon red"><RadioTower size={18} /></span><strong>{data.devices.length - connected}</strong><small>Offline</small></div>
      <div><span className="mapStatIcon blue"><MapPin size={18} /></span><strong>{data.devices.filter((device) => device.location).length}</strong><small>Located</small></div>
      <div><span className="mapStatIcon amber"><Crosshair size={18} /></span><strong>{data.devices.filter((device) => !device.location).length}</strong><small>Unlocated</small></div>
    </section>
    <div className="mapToolbar">
      <div className="segmented" aria-label="Map device filter">
        <button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>All devices</button>
        <button className={filter === "connected" ? "active" : ""} onClick={() => setFilter("connected")}>Online</button>
        <button className={filter === "offline" ? "active" : ""} onClick={() => setFilter("offline")}>Offline</button>
      </div>
      <div className="mapToolbarEnd">
        <span className="mapToolbarNote">Green is online, red is offline. Network-derived positions are approximate.</span>
        <span className="mapProviderBadge"><MapIcon size={14} /> OpenStreetMap vector</span>
      </div>
    </div>
    <section className="fleetMapLayout">
      <div className="fleetMap" aria-label="Device location map">
        <OpenStreetFleetMap devices={located} selected={selected} selectionRevision={selectionRevision} onSelect={selectDevice} onRemote={onRemote} />
        <div className="mapLiveBadge"><i /><span><strong>Live fleet</strong><small>{located.length} mapped device{located.length === 1 ? "" : "s"}</small></span></div>
        <div className="mapLegend"><span><i className="online" />Online</span><span><i className="offline" />Offline</span></div>
      </div>
      <aside className="mapDeviceList">
        <header><div><strong>Fleet locations</strong><span>{located.length} visible on map</span></div><Navigation size={18} /></header>
        <div>{data.devices.filter((device) => filter === "all" || device.presence === filter).map((device) => <button className={selected?.id === device.id ? "mapDevice active" : "mapDevice"} key={device.id} onClick={() => device.location ? selectDevice(device) : setEditing(device)}>
          <i className={device.presence === "connected" ? "online" : "offline"} />
          <span><strong>{device.display_name}</strong><small>{device.location?.label || (device.location ? `${device.location.latitude.toFixed(3)}, ${device.location.longitude.toFixed(3)}` : "Location required")}</small></span>
          <MapPin size={16} />
        </button>)}</div>
        {selected && <footer><div><strong>{selected.display_name}</strong><span>{selected.location?.source} location</span></div><button onClick={() => setEditing(selected)}>Edit location</button><button onClick={() => onRemote(selected)}>SSH</button></footer>}
      </aside>
    </section>
    {editing && <LocationDialog device={editing} data={data} onClose={() => setEditing(null)} onSaved={(device) => { onData({ ...data, devices: data.devices.map((item) => item.id === device.id ? device : item) }); setSelected(device); setEditing(null); }} />}
  </>;
}

function OpenStreetFleetMap({ devices, selected, selectionRevision, onSelect, onRemote }: { devices: Device[]; selected: Device | null; selectionRevision: number; onSelect: (device: Device) => void; onRemote: (device: Device) => void }) {
  const containerRef = React.useRef<HTMLDivElement>(null);
  const mapRef = React.useRef<MapLibreMap | null>(null);
  const markersRef = React.useRef(new Map<string, { marker: Marker; popup: Popup }>());
  const onSelectRef = React.useRef(onSelect);
  const onRemoteRef = React.useRef(onRemote);

  onSelectRef.current = onSelect;
  onRemoteRef.current = onRemote;

  React.useEffect(() => {
    if (!containerRef.current) return;
    const map = new MapLibreMap({
      container: containerRef.current,
      style: OPEN_STREET_MAP_STYLE,
      center: [0, 20],
      zoom: 2,
      minZoom: 2,
      maxZoom: 19,
      attributionControl: false,
      canvasContextAttributes: { antialias: true },
    });
    map.addControl(new NavigationControl({ showCompass: true, showZoom: true, visualizePitch: true }), "bottom-right");
    map.addControl(new AttributionControl({
      compact: true,
      customAttribution: '<a href="https://www.openstreetmap.org/copyright" target="_blank">© OpenStreetMap contributors</a> · <a href="https://openfreemap.org" target="_blank">OpenFreeMap</a>',
    }), "bottom-left");
    mapRef.current = map;
    return () => {
      markersRef.current.clear();
      mapRef.current = null;
      map.remove();
    };
  }, []);

  React.useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    markersRef.current.forEach(({ marker, popup }) => {
      popup.remove();
      marker.remove();
    });
    markersRef.current.clear();
    devices.forEach((device) => {
      const element = createMarkerElement(device);
      const popup = new Popup({ offset: 30, closeButton: true, maxWidth: "280px" })
        .setDOMContent(createPopupContent(device, () => onRemoteRef.current(device)));
      const marker = new Marker({ element, anchor: "bottom" })
        .setLngLat([device.location!.longitude, device.location!.latitude])
        .addTo(map);
      element.addEventListener("click", (event) => {
        event.stopPropagation();
        onSelectRef.current(device);
        if (!popup.isOpen()) popup.setLngLat(marker.getLngLat()).addTo(map);
      });
      markersRef.current.set(device.id, { marker, popup });
    });
    if (devices.length === 1) {
      map.easeTo({ center: [devices[0].location!.longitude, devices[0].location!.latitude], zoom: 11, duration: 700 });
    } else {
      if (!devices.length) return;
      const bounds = new LngLatBounds();
      devices.forEach((device) => bounds.extend([device.location!.longitude, device.location!.latitude]));
      map.fitBounds(bounds, { padding: 72, maxZoom: 12, duration: 700 });
    }
  }, [devices]);

  React.useEffect(() => {
    if (!selected?.location) return;
    const map = mapRef.current;
    const record = markersRef.current.get(selected.id);
    if (!map || !record) return;
    map.easeTo({ center: [selected.location.longitude, selected.location.latitude], zoom: Math.max(map.getZoom(), 11), duration: 650 });
    if (!record.popup.isOpen()) record.popup.setLngLat(record.marker.getLngLat()).addTo(map);
  }, [selected, selectionRevision]);

  return <div ref={containerRef} className="maplibreMap" />;
}

function LocationDialog({ device, data, onClose, onSaved }: { device: Device; data: FleetData; onClose: () => void; onSaved: (device: Device) => void }) {
  const [latitude, setLatitude] = React.useState(String(device.location?.latitude ?? ""));
  const [longitude, setLongitude] = React.useState(String(device.location?.longitude ?? ""));
  const [label, setLabel] = React.useState(device.location?.label || "");
  const [error, setError] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  async function save() {
    const lat = Number(latitude); const lon = Number(longitude);
    if (!Number.isFinite(lat) || lat < -90 || lat > 90 || !Number.isFinite(lon) || lon < -180 || lon > 180) { setError("Enter valid latitude and longitude values."); return; }
    setSaving(true); setError("");
    try { onSaved(await updateDevice(data.membership, device.id, { location: { latitude: lat, longitude: lon, label, source: "manual" } })); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Location could not be saved."); }
    finally { setSaving(false); }
  }
  return <div className="overlay centered" onMouseDown={onClose}><section className="modal locationModal" onMouseDown={(event) => event.stopPropagation()}><header><div><span className="overline">Device position</span><h2>Set {device.display_name} location</h2></div><button className="iconButton" onClick={onClose}><X size={18} /></button></header><div className="modalForm"><div className="coordinateFields"><label>Latitude<input inputMode="decimal" value={latitude} onChange={(event) => setLatitude(event.target.value)} placeholder="18.5204" /></label><label>Longitude<input inputMode="decimal" value={longitude} onChange={(event) => setLongitude(event.target.value)} placeholder="73.8567" /></label></div><label>Site label<input value={label} onChange={(event) => setLabel(event.target.value)} placeholder="Pune lab" /></label>{error && <div className="formError">{error}</div>}<p className="formHint">Use GPSD for precise mobile coordinates, static coordinates for a fixed site, or opt-in IP location for an approximate network position.</p><div className="modalActions"><button className="button secondary" onClick={onClose}>Cancel</button><button className="button successButton" disabled={saving} onClick={save}><Save size={16} />{saving ? "Saving..." : "Save location"}</button></div></div></section></div>;
}

function createMarkerElement(device: Device) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "mapMarkerShell";
  button.setAttribute("aria-label", `${device.display_name}, ${device.presence}`);
  const pin = document.createElement("span");
  pin.className = `mapMarker ${device.presence === "connected" ? "online" : "offline"}`;
  pin.appendChild(document.createElement("i"));
  button.appendChild(pin);
  return button;
}

function createPopupContent(device: Device, onRemote: () => void) {
  const content = document.createElement("div");
  content.className = "mapPopup";
  const name = document.createElement("strong");
  name.textContent = device.display_name;
  const location = document.createElement("span");
  location.textContent = device.location?.label || `${device.location?.latitude.toFixed(4)}, ${device.location?.longitude.toFixed(4)}`;
  const state = document.createElement("small");
  state.textContent = `${device.presence} · ${device.location?.source}`;
  const remote = document.createElement("button");
  remote.type = "button";
  remote.textContent = "Remote access";
  remote.addEventListener("click", onRemote);
  content.append(name, location, state, remote);
  return content;
}
