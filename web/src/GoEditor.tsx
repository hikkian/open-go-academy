import { useEffect, useRef } from 'react';
import { basicSetup, EditorView } from 'codemirror';
import { autocompletion } from '@codemirror/autocomplete';
import { indentWithTab } from '@codemirror/commands';
import { go } from '@codemirror/lang-go';
import { HighlightStyle, indentUnit, syntaxHighlighting } from '@codemirror/language';
import { Compartment, EditorState } from '@codemirror/state';
import { keymap } from '@codemirror/view';
import { tags } from '@lezer/highlight';

const goHighlight = HighlightStyle.define([
  { tag: [tags.keyword, tags.controlKeyword], color: '#c5a5ff' },
  { tag: [tags.string, tags.character], color: '#f6b26b' },
  { tag: [tags.number, tags.bool, tags.atom], color: '#7dd3fc' },
  { tag: tags.comment, color: '#aab9c9' },
  { tag: [tags.typeName, tags.className, tags.namespace], color: '#86dbab' },
  { tag: tags.function(tags.variableName), color: '#f5ca7c' },
  { tag: [tags.variableName, tags.propertyName], color: '#d9e6ef' },
  { tag: [tags.operator, tags.punctuation], color: '#85d8d2' },
]);

export default function GoEditor({ value, onChange, readOnly = false, scrollLabel = 'Прокрутка кода Go' }: { value: string; onChange: (value: string) => void; readOnly?: boolean; scrollLabel?: string }) {
  const host = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const changeRef = useRef(onChange);
 const readonlyConfig=useRef(new Compartment());
 const externalChange=useRef(false);
  changeRef.current = onChange;
  useEffect(() => {
    if (!host.current) return;
    const view = new EditorView({ doc: value, extensions: [basicSetup, autocompletion({ activateOnTyping: false }), keymap.of([indentWithTab]), indentUnit.of('\t'), EditorState.tabSize.of(4), go(), syntaxHighlighting(goHighlight), readonlyConfig.current.of([EditorState.readOnly.of(readOnly), EditorView.editable.of(!readOnly)]), EditorView.lineWrapping, EditorView.contentAttributes.of({ 'aria-label': 'Go-код' }), EditorView.updateListener.of(update => { if (update.docChanged&&!externalChange.current) changeRef.current(update.state.doc.toString()); })], parent: host.current });
    view.scrollDOM.tabIndex = 0;
    view.scrollDOM.setAttribute('role', 'region');
    view.scrollDOM.setAttribute('aria-label', scrollLabel);
    viewRef.current = view;
    return () => { view.destroy(); viewRef.current = null; };
  }, []);
  useEffect(() => { viewRef.current?.scrollDOM.setAttribute('aria-label', scrollLabel); }, [scrollLabel]);
 useEffect(()=>{viewRef.current?.dispatch({effects:readonlyConfig.current.reconfigure([EditorState.readOnly.of(readOnly),EditorView.editable.of(!readOnly)])});},[readOnly]);
  useEffect(() => {
    const view = viewRef.current;
    if (view && view.state.doc.toString() !== value) {externalChange.current=true;try{view.dispatch({changes:{from:0,to:view.state.doc.length,insert:value}})}finally{externalChange.current=false}}
  }, [value]);
  return <div className="editor-canvas" ref={host} aria-label="Редактор Go" />;
}
