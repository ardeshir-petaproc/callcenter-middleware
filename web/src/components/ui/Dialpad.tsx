const KEYS: [string, string][] = [
  ['1', ''], ['2', 'ABC'], ['3', 'DEF'], ['4', 'GHI'], ['5', 'JKL'], ['6', 'MNO'],
  ['7', 'PQRS'], ['8', 'TUV'], ['9', 'WXYZ'], ['*', ''], ['0', '+'], ['#', ''],
];

export function Dialpad({ onPress }: { onPress: (k: string) => void }) {
  return (
    <div className="dialpad" role="group" aria-label="Dial pad">
      {KEYS.map(([k, sub]) => (
        <button key={k} type="button" onClick={() => onPress(k)} aria-label={`Key ${k}`}>
          {k}{sub && <small>{sub}</small>}
        </button>
      ))}
    </div>
  );
}
