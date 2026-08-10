import React from "react";
import L from "leaflet";
import { MapContainer, Marker, Popup, TileLayer, useMap } from "react-leaflet";
import { Crosshair, MapPin, Navigation, RadioTower, Save, X } from "lucide-react";
import "leaflet/dist/leaflet.css";
import { Device, FleetData, updateDevice } from "./api";

type Props = { data: FleetData; onData: (data: FleetData) => void; onRemote: (device: Device) => void };

export function FleetMap({ data, onData, onRemote }: Props) {
  const [filter, setFilter] = React.useState<"all" | "connected" | "offline">("all");
  const [editing, setEditing] = React.useState<Device | null>(null);
  const [selected, setSelected] = React.useState<Device | null>(null);
  const located = data.devices.filter((device) => device.location && (filter === "all" || device.presence === filter));
  const connected = data.devices.filter((device) => device.presence === "connected").length;
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
      <span>Green is online, red is offline. Coordinates are never synthesized.</span>
    </div>
    <section className="fleetMapLayout">
      <div className="fleetMap" aria-label="Device location map">
        <MapContainer center={[20, 0]} zoom={2} minZoom={2} scrollWheelZoom className="leafletMap">
          <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://tile.openstreetmap.org/{z}/{x}/{y}.png" />
          <FitDevices devices={located} />
          {located.map((device) => <Marker key={device.id} position={[device.location!.latitude, device.location!.longitude]} icon={markerIcon(device.presence)} eventHandlers={{ click: () => setSelected(device) }}>
            <Popup><div className="mapPopup"><strong>{device.display_name}</strong><span>{device.location?.label || `${device.location?.latitude.toFixed(4)}, ${device.location?.longitude.toFixed(4)}`}</span><small>{device.presence} · {device.location?.source}</small><button onClick={() => onRemote(device)}>Remote access</button></div></Popup>
          </Marker>)}
        </MapContainer>
      </div>
      <aside className="mapDeviceList">
        <header><div><strong>Fleet locations</strong><span>{located.length} visible on map</span></div><Navigation size={18} /></header>
        <div>{data.devices.filter((device) => filter === "all" || device.presence === filter).map((device) => <button className={selected?.id === device.id ? "mapDevice active" : "mapDevice"} key={device.id} onClick={() => device.location ? setSelected(device) : setEditing(device)}>
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

function FitDevices({ devices }: { devices: Device[] }) {
  const map = useMap();
  React.useEffect(() => {
    if (!devices.length) return;
    const bounds = L.latLngBounds(devices.map((device) => [device.location!.latitude, device.location!.longitude]));
    map.fitBounds(bounds, { padding: [48, 48], maxZoom: 12 });
  }, [devices, map]);
  return null;
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
  return <div className="overlay centered" onMouseDown={onClose}><section className="modal locationModal" onMouseDown={(event) => event.stopPropagation()}><header><div><span className="overline">Actual device position</span><h2>Set {device.display_name} location</h2></div><button className="iconButton" onClick={onClose}><X size={18} /></button></header><div className="modalForm"><div className="coordinateFields"><label>Latitude<input inputMode="decimal" value={latitude} onChange={(event) => setLatitude(event.target.value)} placeholder="18.5204" /></label><label>Longitude<input inputMode="decimal" value={longitude} onChange={(event) => setLongitude(event.target.value)} placeholder="73.8567" /></label></div><label>Site label<input value={label} onChange={(event) => setLabel(event.target.value)} placeholder="Pune lab" /></label>{error && <div className="formError">{error}</div>}<p className="formHint">Use GPSD in the agent configuration for automatic coordinates, or enter the installed site position here.</p><div className="modalActions"><button className="button secondary" onClick={onClose}>Cancel</button><button className="button successButton" disabled={saving} onClick={save}><Save size={16} />{saving ? "Saving..." : "Save location"}</button></div></div></section></div>;
}

function markerIcon(presence: string) {
  return L.divIcon({ className: "mapMarkerShell", html: `<span class="mapMarker ${presence === "connected" ? "online" : "offline"}"><i></i></span>`, iconSize: [30, 38], iconAnchor: [15, 36], popupAnchor: [0, -34] });
}
